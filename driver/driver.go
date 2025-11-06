// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/drivers/shared/eventer"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	"github.com/hashicorp/nomad/plugins/shared/structs"
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
		SendSignals: true,
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
	// TODO: Initialize this once daemon API is available
	daemonClient DaemonClient

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

	// TODO: Initialize gRPC client to Elide daemon once API is available
	// This will connect to the daemon socket/address and prepare for snippet execution
	// Example:
	//   client, err := NewDaemonClient(d.config.DaemonSocket, d.config.DaemonAddress)
	//   if err != nil {
	//       return fmt.Errorf("failed to connect to Elide daemon: %v", err)
	//   }
	//   d.daemonClient = client

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

	// TODO: Check if Elide daemon is available/running
	// This could check:
	// - If daemon socket exists and is accessible
	// - If daemon responds to health check
	// - If Elide binary is available (if auto_start_daemon is enabled)

	// For now, report as healthy if no error
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
		return nil, nil, fmt.Errorf("failed to decode driver config: %v", err)
	}

	if err := taskConfig.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid task config: %v", err)
	}

	d.logger.Info("starting task", "task_id", cfg.ID, "language", taskConfig.Language)

	_ = drivers.NewTaskHandle(taskHandleVersion)
	_ = cfg

	// TODO: Once daemon API is available, this becomes a simple gRPC call:
	//
	// 1. Ensure daemon is running (if auto_start_daemon is enabled)
	//    if err := d.ensureDaemonRunning(); err != nil {
	//        return nil, nil, fmt.Errorf("failed to start daemon: %v", err)
	//    }
	//
	// 2. Read script code (either from file or use inline code)
	//    code, err := d.readScriptCode(&taskConfig, cfg.TaskDir().Dir)
	//    if err != nil {
	//        return nil, nil, fmt.Errorf("failed to read script: %v", err)
	//    }
	//
	// 3. Call ExecuteSnippet gRPC
	//    resp, err := d.daemonClient.ExecuteSnippet(context.Background(), &ExecuteSnippetRequest{
	//        Code:        code,
	//        Language:    taskConfig.Language,
	//        Env:         taskConfig.Env,
	//        ExecutionId: cfg.ID,
	//        Args:        taskConfig.Args,
	//    })
	//    if err != nil {
	//        return nil, nil, fmt.Errorf("failed to execute snippet: %v", err)
	//    }
	//
	// 4. Create task handle with execution ID
	//    h := &taskHandle{
	//        executionId: resp.ExecutionId,
	//        taskConfig:  cfg,
	//        startedAt:    time.Now(),
	//        status:      resp.Status,
	//        logger:      d.logger.With("task_id", cfg.ID),
	//    }
	//
	// 5. Store handle and return
	//    driverState := TaskState{
	//        ExecutionId: resp.ExecutionId,
	//        TaskConfig:  cfg,
	//        StartedAt:   h.startedAt,
	//    }
	//    if err := handle.SetDriverState(&driverState); err != nil {
	//        return nil, nil, fmt.Errorf("failed to set driver state: %v", err)
	//    }
	//    d.tasks.Set(cfg.ID, h)
	//    return handle, nil, nil

	// TEMPORARY: Return error until daemon API is available
	return nil, nil, fmt.Errorf("daemon API not yet available - waiting for Elide team to implement ExecuteSnippet gRPC")
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
		return fmt.Errorf("failed to decode task state from handle: %v", err)
	}

	_ = taskState // TODO: Use once daemon API is available

	// TODO: Once daemon API is available, reattach to execution:
	//
	// 1. Ensure daemon client is connected
	//    if d.daemonClient == nil {
	//        client, err := NewDaemonClient(d.config.DaemonSocket, d.config.DaemonAddress)
	//        if err != nil {
	//            return fmt.Errorf("failed to reconnect to daemon: %v", err)
	//        }
	//        d.daemonClient = client
	//    }
	//
	// 2. Check execution status
	//    status, err := d.daemonClient.GetExecutionStatus(context.Background(), taskState.ExecutionId)
	//    if err != nil {
	//        return fmt.Errorf("failed to get execution status: %v", err)
	//    }
	//
	// 3. Recreate handle
	//    h := &taskHandle{
	//        executionId: taskState.ExecutionId,
	//        taskConfig:  taskState.TaskConfig,
	//        startedAt:   taskState.StartedAt,
	//        status:      status,
	//        logger:      d.logger.With("task_id", taskState.TaskConfig.ID),
	//    }
	//    d.tasks.Set(taskState.TaskConfig.ID, h)
	//    return nil

	return fmt.Errorf("daemon API not yet available - cannot recover task")
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

	_ = handle // TODO: Use once daemon API is available

	// TODO: Once daemon API is available, poll for execution completion:
	//
	// ticker := time.NewTicker(1 * time.Second)
	// defer ticker.Stop()
	//
	// for {
	//     select {
	//     case <-ctx.Done():
	//         return
	//     case <-d.ctx.Done():
	//         return
	//     case <-ticker.C:
	//         status, err := d.daemonClient.GetExecutionStatus(ctx, handle.executionId)
	//         if err != nil {
	//             ch <- &drivers.ExitResult{
	//                 Err: fmt.Errorf("failed to get execution status: %v", err),
	//             }
	//             return
	//         }
	//
	//         if status.Complete {
	//             result := &drivers.ExitResult{
	//                 ExitCode: status.ExitCode,
	//             }
	//             if status.Error != "" {
	//                 result.Err = errors.New(status.Error)
	//             }
	//             ch <- result
	//             return
	//         }
	//     }
	// }

	// TEMPORARY: Return error until daemon API is available
	ch <- &drivers.ExitResult{
		Err: fmt.Errorf("daemon API not yet available"),
	}
}

// StopTask stops a running task with the given signal and within the timeout window.
func (d *ElideDriverPlugin) StopTask(taskID string, timeout time.Duration, signal string) error {
	_, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	_ = timeout  // TODO: Use once daemon API is available
	_ = signal   // TODO: Use once daemon API is available

	// TODO: Once daemon API is available, call CancelExecution:
	//
	// ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel()
	//
	// _, err := d.daemonClient.CancelExecution(ctx, &CancelExecutionRequest{
	//     ExecutionId: handle.executionId,
	// })
	// if err != nil {
	//     return fmt.Errorf("failed to cancel execution: %v", err)
	// }
	//
	// return nil

	return fmt.Errorf("daemon API not yet available - cannot stop task")
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

