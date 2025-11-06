// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/drivers/shared/eventer"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	"github.com/hashicorp/nomad/plugins/shared/structs"

	pb "github.com/elide-dev/elide-task-driver/proto/gen/go/elide/daemon/v1alpha1"
)

const (
	// pluginName is the name of the plugin
	pluginName = "elide"

	// pluginVersion allows the client to identify and use newer versions of
	// an installed plugin
	pluginVersion = "v0.1.0"

	// fingerprintPeriod is the interval at which the plugin will send
	// fingerprint responses
	fingerprintPeriod = 30 * time.Second

	// taskHandleVersion is the version of task handle which this plugin sets
	// and understands how to decode
	taskHandleVersion = 1

	// executeSnippetTimeout is the default timeout for ExecuteSnippet RPCs.
	executeSnippetTimeout = 10 * time.Second

	// statusRequestTimeout is the default timeout for status polling RPCs.
	statusRequestTimeout = 5 * time.Second
)

var (
	// pluginInfo describes the plugin
	pluginInfo = &base.PluginInfoResponse{
		Type:              base.PluginTypeDriver,
		PluginApiVersions: []string{drivers.ApiVersion010},
		PluginVersion:     pluginVersion,
		Name:              pluginName,
	}

	// capabilities indicates what optional features this driver supports
	capabilities = &drivers.Capabilities{
		SendSignals: false,
		Exec:        false, // Not implementing exec for MVP
		FSIsolation: drivers.FSIsolationNone,
		NetIsolationModes: []drivers.NetIsolationMode{
			drivers.NetIsolationModeHost,
		},
	}
)

// ElideDriverPlugin is the Nomad task driver plugin for Elide runtime
type ElideDriverPlugin struct {
	// eventer is used to handle multiplexing of TaskEvents calls
	eventer *eventer.Eventer

	// config is the plugin configuration set by the SetConfig RPC
	config *Config

	// nomadConfig is the client config from Nomad
	nomadConfig *base.ClientDriverConfig

	// tasks is the in memory datastore mapping taskIDs to driver handles
	tasks *taskStore

	// daemonClient is the gRPC client to the Elide daemon
	daemonClient DaemonClient

	// sessionID is the session ID for this Nomad client (one session per client)
	sessionID string

	// sessionLock serializes session initialization.
	sessionLock sync.Mutex

	// ctx is the context for the driver
	ctx context.Context

	// signalShutdown is called when the driver is shutting down
	signalShutdown context.CancelFunc

	// logger will log to the Nomad agent
	logger hclog.Logger
}

// NewPlugin returns a new Elide driver plugin
func NewPlugin(logger hclog.Logger) drivers.DriverPlugin {
	ctx, cancel := context.WithCancel(context.Background())
	logger = logger.Named(pluginName)

	return &ElideDriverPlugin{
		eventer:        eventer.NewEventer(ctx, logger),
		config:         &Config{},
		tasks:          newTaskStore(),
		ctx:            ctx,
		signalShutdown: cancel,
		logger:         logger,
	}
}

// PluginInfo returns information describing the plugin.
func (d *ElideDriverPlugin) PluginInfo() (*base.PluginInfoResponse, error) {
	return pluginInfo, nil
}

// ConfigSchema returns the plugin configuration schema.
func (d *ElideDriverPlugin) ConfigSchema() (*hclspec.Spec, error) {
	return configSpec, nil
}

// SetConfig is called by the client to pass the configuration for the plugin.
func (d *ElideDriverPlugin) SetConfig(cfg *base.Config) error {
	var config Config
	if len(cfg.PluginConfig) != 0 {
		if err := base.MsgPackDecode(cfg.PluginConfig, &config); err != nil {
			return err
		}
	}

	// Save the configuration to the plugin
	d.config = &config

	// Save the Nomad agent configuration
	if cfg.AgentConfig != nil {
		d.nomadConfig = cfg.AgentConfig.Driver
	}

	// Initialize gRPC client to Elide daemon
	client, err := NewDaemonClient(d.config.DaemonSocket, d.config.DaemonAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to Elide daemon: %w", err)
	}
	d.daemonClient = client

	// Check daemon health
	if err := d.daemonClient.Health(context.Background()); err != nil {
		d.logger.Warn("daemon health check failed", "error", err)
	}

	// Ensure session exists (one session per Nomad client)
	if err := d.ensureSession(context.Background()); err != nil {
		// Don't fail SetConfig if daemon isn't available yet.
		// The fingerprint will report it as undetected.
		d.logger.Warn("failed to initialize session (daemon may not be running yet)", "error", err)
	}

	return nil
}

// TaskConfigSchema returns the HCL schema for the configuration of a task.
func (d *ElideDriverPlugin) TaskConfigSchema() (*hclspec.Spec, error) {
	return taskConfigSpec, nil
}

// Capabilities returns the features supported by the driver.
func (d *ElideDriverPlugin) Capabilities() (*drivers.Capabilities, error) {
	return capabilities, nil
}

// Fingerprint returns a channel that will be used to send health information
// and other driver specific node attributes.
func (d *ElideDriverPlugin) Fingerprint(ctx context.Context) (<-chan *drivers.Fingerprint, error) {
	ch := make(chan *drivers.Fingerprint)
	go d.handleFingerprint(ctx, ch)
	return ch, nil
}

// handleFingerprint manages the channel and the flow of fingerprint data.
func (d *ElideDriverPlugin) handleFingerprint(ctx context.Context, ch chan<- *drivers.Fingerprint) {
	defer close(ch)

	// Nomad expects the initial fingerprint to be sent immediately
	ticker := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			ticker.Reset(fingerprintPeriod)
			ch <- d.buildFingerprint()
		}
	}
}

// buildFingerprint returns the driver's fingerprint data
func (d *ElideDriverPlugin) buildFingerprint() *drivers.Fingerprint {
	fp := &drivers.Fingerprint{
		Attributes:        map[string]*structs.Attribute{},
		Health:            drivers.HealthStateHealthy,
		HealthDescription: drivers.DriverHealthy,
	}

	// Check if Elide daemon is available/running
	socketPath := d.config.DaemonSocket
	if socketPath == "" {
		socketPath = "/tmp/elide-daemon.sock"
	}

	// Check if socket exists
	if _, err := os.Stat(socketPath); err != nil {
		fp.Health = drivers.HealthStateUndetected
		fp.HealthDescription = fmt.Sprintf("daemon socket not found: %s", socketPath)
		return fp
	}

	// If daemon client is available, check health
	if d.daemonClient != nil {
		if err := d.daemonClient.Health(context.Background()); err != nil {
			fp.Health = drivers.HealthStateUnhealthy
			fp.HealthDescription = fmt.Sprintf("daemon health check failed: %v", err)
			return fp
		}
	}

	// Report driver as available
	fp.Attributes["driver.elide.available"] = structs.NewBoolAttribute(true)
	if d.sessionID != "" {
		fp.Attributes["driver.elide.session_id"] = structs.NewStringAttribute(d.sessionID)
	}

	return fp
}

// StartTask returns a task handle and a driver network if necessary.
// This will be simplified to a gRPC call once daemon API is available.
func (d *ElideDriverPlugin) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
	if _, ok := d.tasks.Get(cfg.ID); ok {
		return nil, nil, fmt.Errorf("task with ID %q already started", cfg.ID)
	}

	var taskConfig TaskConfig
	if err := cfg.DecodeDriverConfig(&taskConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to decode driver config: %w", err)
	}

	if err := taskConfig.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid task config: %w", err)
	}

	// Validate language against session's enabled languages
	enabledLanguages := d.config.SessionConfig.EnabledLanguages
	if len(enabledLanguages) == 0 {
		enabledLanguages = []string{"python", "javascript", "typescript"} // defaults
	}
	if err := taskConfig.ValidateLanguage(enabledLanguages); err != nil {
		return nil, nil, fmt.Errorf("language validation failed: %w", err)
	}

	d.logger.Info("starting task", "task_id", cfg.ID, "language", taskConfig.Language)

	// Ensure session exists before starting task
	if err := d.ensureSession(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("failed to ensure session: %w", err)
	}

	// Read script code (either from file or use inline code)
	var code string
	var err error
	if taskConfig.Code != "" {
		code = taskConfig.Code
	} else if taskConfig.Script != "" {
		baseDir := filepath.Clean(cfg.TaskDir().Dir)
		scriptPath := filepath.Clean(filepath.Join(baseDir, taskConfig.Script))
		if !strings.HasPrefix(scriptPath, baseDir+string(os.PathSeparator)) && scriptPath != baseDir {
			return nil, nil, fmt.Errorf("script path %q escapes task directory", taskConfig.Script)
		}
		codeBytes, err := os.ReadFile(scriptPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read script file: %w", err)
		}
		code = string(codeBytes)
	} else {
		return nil, nil, fmt.Errorf("either 'script' or 'code' must be specified")
	}

	// Call ExecuteSnippet gRPC within session
	execCtx, cancel := d.withTimeout(context.Background(), executeSnippetTimeout)
	defer cancel()

	resp, err := d.daemonClient.ExecuteSnippet(
		execCtx,
		d.sessionID,
		cfg.ID,
		code,
		taskConfig.Language,
		taskConfig.Env,
		taskConfig.Args,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute snippet: %w", err)
	}

	// Create task handle
	handle := drivers.NewTaskHandle(taskHandleVersion)
	handle.Config = cfg

	h := &taskHandle{
		executionId: resp.ExecutionId,
		sessionId:   d.sessionID,
		taskConfig:  cfg,
		startedAt:   time.Now(),
		status:      resp.Status.String(),
		logger:      d.logger.With("task_id", cfg.ID),
	}

	// Store handle and return
	driverState := TaskState{
		ExecutionId: resp.ExecutionId,
		SessionId:   d.sessionID,
		TaskConfig:  cfg,
		StartedAt:   h.startedAt,
	}
	if err := handle.SetDriverState(&driverState); err != nil {
		return nil, nil, fmt.Errorf("failed to set driver state: %w", err)
	}
	d.tasks.Set(cfg.ID, h)

	d.logger.Info("task started", "task_id", cfg.ID, "execution_id", resp.ExecutionId, "session_id", d.sessionID)
	return handle, nil, nil
}

// RecoverTask recreates the in-memory state of a task from a TaskHandle.
func (d *ElideDriverPlugin) RecoverTask(handle *drivers.TaskHandle) error {
	if handle == nil {
		return errors.New("handle cannot be nil")
	}

	if _, ok := d.tasks.Get(handle.Config.ID); ok {
		return nil
	}

	var taskState TaskState
	if err := handle.GetDriverState(&taskState); err != nil {
		return fmt.Errorf("failed to decode task state from handle: %w", err)
	}

	// Ensure daemon client is connected
	if d.daemonClient == nil {
		client, err := NewDaemonClient(d.config.DaemonSocket, d.config.DaemonAddress)
		if err != nil {
			return fmt.Errorf("failed to reconnect to daemon: %w", err)
		}
		d.daemonClient = client
		d.sessionID = taskState.SessionId
	}

	// Check execution status
	statusCtx, cancel := d.withTimeout(context.Background(), statusRequestTimeout)
	statusResp, err := d.daemonClient.GetExecutionStatus(statusCtx, taskState.SessionId, taskState.ExecutionId)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to get execution status: %w", err)
	}

	// Recreate handle
	h := &taskHandle{
		executionId: taskState.ExecutionId,
		sessionId:   taskState.SessionId,
		taskConfig:  taskState.TaskConfig,
		startedAt:   taskState.StartedAt,
		status:      statusResp.Status.String(),
		logger:      d.logger.With("task_id", taskState.TaskConfig.ID),
	}

	// If execution is complete, set exit result
	if statusResp.Complete {
		result := &drivers.ExitResult{
			ExitCode: int(statusResp.ExitCode),
		}
		if statusResp.Error != "" {
			result.Err = errors.New(statusResp.Error)
		}
		h.SetCompleted(result)
	}

	d.tasks.Set(taskState.TaskConfig.ID, h)
	return nil
}

// WaitTask returns a channel used to notify Nomad when a task exits.
func (d *ElideDriverPlugin) WaitTask(ctx context.Context, taskID string) (<-chan *drivers.ExitResult, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	ch := make(chan *drivers.ExitResult)
	go d.handleWait(ctx, handle, ch)
	return ch, nil
}

func (d *ElideDriverPlugin) handleWait(ctx context.Context, handle *taskHandle, ch chan *drivers.ExitResult) {
	defer close(ch)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			statusCtx, cancel := d.withTimeout(ctx, statusRequestTimeout)
			statusResp, err := d.daemonClient.GetExecutionStatus(statusCtx, handle.sessionId, handle.executionId)
			cancel()
			if err != nil {
				ch <- &drivers.ExitResult{
					Err: fmt.Errorf("failed to get execution status: %w", err),
				}
				return
			}

			// Update handle status
			handle.stateLock.Lock()
			handle.status = statusResp.Status.String()
			handle.stateLock.Unlock()

			if statusResp.Complete {
				result := &drivers.ExitResult{
					ExitCode: int(statusResp.ExitCode),
				}
				if statusResp.Error != "" {
					result.Err = errors.New(statusResp.Error)
				}
				handle.SetCompleted(result)
				ch <- result
				return
			}
		}
	}
}

// StopTask stops a running task with the given signal and within the timeout window.
func (d *ElideDriverPlugin) StopTask(taskID string, timeout time.Duration, signal string) error {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	_ = signal // TODO: Forward signal to execution if daemon supports it

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := d.daemonClient.CancelExecution(ctx, handle.sessionId, handle.executionId)
	if err != nil {
		return fmt.Errorf("failed to cancel execution: %w", err)
	}

	return nil
}

// DestroyTask cleans up and removes a task that has terminated.
func (d *ElideDriverPlugin) DestroyTask(taskID string, force bool) error {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil
	}

	if handle.IsRunning() && !force {
		return errors.New("cannot destroy running task")
	}

	_ = handle // TODO: Use once daemon API is available for cleanup

	// TODO: Cleanup any resources if needed
	// The daemon should handle cleanup automatically when execution completes

	d.tasks.Delete(taskID)
	return nil
}

// InspectTask returns detailed status information for the referenced taskID.
func (d *ElideDriverPlugin) InspectTask(taskID string) (*drivers.TaskStatus, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	return handle.TaskStatus(), nil
}

// TaskStats returns a channel which the driver should send stats to at the given interval.
func (d *ElideDriverPlugin) TaskStats(ctx context.Context, taskID string, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	_, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	_ = interval // TODO: Use once daemon API is available

	// TODO: Once daemon API is available, collect stats from daemon:
	// This could query daemon for resource usage per execution
	// For now, return empty stats channel

	ch := make(chan *drivers.TaskResourceUsage)
	go func() {
		defer close(ch)
		// Empty stats for now
		<-ctx.Done()
	}()

	return ch, nil
}

// TaskEvents returns a channel that the plugin can use to emit task related events.
func (d *ElideDriverPlugin) TaskEvents(ctx context.Context) (<-chan *drivers.TaskEvent, error) {
	return d.eventer.TaskEvents(ctx)
}

// SignalTask forwards a signal to a task.
func (d *ElideDriverPlugin) SignalTask(taskID string, signal string) error {
	_, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	_ = signal // TODO: Use once daemon API is available

	// TODO: Once daemon API is available, forward signal via CancelExecution:
	// The signal could be passed as a parameter to cancellation

	d.logger.Warn("signal forwarding not yet implemented", "task_id", taskID, "signal", signal)
	return nil
}

// ExecTask returns the result of executing the given command inside a task.
// This is not supported for MVP.
func (d *ElideDriverPlugin) ExecTask(taskID string, cmd []string, timeout time.Duration) (*drivers.ExecTaskResult, error) {
	return nil, errors.New("exec not supported")
}

// Shutdown is called when the driver is being shut down and should
// clean up any resources, including closing the session with the daemon.
func (d *ElideDriverPlugin) Shutdown() {
	d.logger.Info("shutting down elide driver")

	// Clean up session with daemon
	if d.sessionID != "" && d.daemonClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		d.logger.Info("deleting session", "session_id", d.sessionID)
		if err := d.daemonClient.DeleteSession(ctx, d.sessionID); err != nil {
			d.logger.Warn("failed to delete session on shutdown", "error", err, "session_id", d.sessionID)
		} else {
			d.logger.Info("session deleted successfully", "session_id", d.sessionID)
		}
	}

	// Close daemon client connection
	if d.daemonClient != nil {
		if err := d.daemonClient.Close(); err != nil {
			d.logger.Warn("failed to close daemon client", "error", err)
		}
	}

	// Signal shutdown to all goroutines
	d.signalShutdown()
}

func (d *ElideDriverPlugin) withTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}

	if timeout <= 0 {
		return context.WithCancel(parent)
	}

	return context.WithTimeout(parent, timeout)
}

func (d *ElideDriverPlugin) ensureSession(ctx context.Context) error {
	if d.daemonClient == nil {
		return errors.New("daemon client not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	if d.sessionID != "" {
		return nil
	}

	sessionID := d.generateSessionID()
	sessionConfig := d.buildSessionConfig()

	const attempts = 5
	baseDelay := 200 * time.Millisecond
	var lastErr error

	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		createCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		resp, err := d.daemonClient.CreateSession(createCtx, sessionID, sessionConfig)
		cancel()
		if err == nil && resp != nil {
			d.sessionID = resp.SessionId
			d.logger.Info("created session", "session_id", d.sessionID, "attempt", i+1)
			return nil
		}
		if err != nil {
			lastErr = err
		}

		getCtx, getCancel := context.WithTimeout(ctx, 3*time.Second)
		getResp, getErr := d.daemonClient.GetSession(getCtx, sessionID)
		getCancel()
		if getErr == nil && getResp != nil {
			d.sessionID = getResp.SessionId
			d.logger.Info("reusing existing session", "session_id", d.sessionID)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * baseDelay):
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to create session after retries: %w", lastErr)
	}
	return errors.New("failed to create or reuse session: unknown error")
}

func (d *ElideDriverPlugin) buildSessionConfig() *pb.SessionConfiguration {
	contextPoolSize := d.config.SessionConfig.ContextPoolSize
	if contextPoolSize == 0 {
		contextPoolSize = 10
	}

	enabledLanguages := d.config.SessionConfig.EnabledLanguages
	if len(enabledLanguages) == 0 {
		enabledLanguages = []string{"python", "javascript", "typescript"}
	}

	enabledIntrinsics := d.config.SessionConfig.EnabledIntrinsics
	if len(enabledIntrinsics) == 0 {
		enabledIntrinsics = []string{"io", "env"}
	}

	memoryLimitMB := d.config.SessionConfig.MemoryLimitMB
	if memoryLimitMB == 0 {
		memoryLimitMB = 512
	}

	return &pb.SessionConfiguration{
		ContextPoolSize:   uint32(contextPoolSize),
		EnabledLanguages:  enabledLanguages,
		EnabledIntrinsics: enabledIntrinsics,
		MemoryLimitMb:     uint64(memoryLimitMB),
		EnableAi:          d.config.SessionConfig.EnableAI,
	}
}

func (d *ElideDriverPlugin) generateSessionID() string {
	if d.sessionID != "" {
		return d.sessionID
	}

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}

	return fmt.Sprintf("nomad-%s", hostname)
}
