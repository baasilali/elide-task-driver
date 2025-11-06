// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"context"
	"errors"
	"time"

	"github.com/elide-dev/elide-task-driver/driver"
	pb "github.com/elide-dev/elide-task-driver/proto/gen/go/elide/daemon/v1alpha1"
)

// MockDaemonClient is a mock implementation of DaemonClient for testing
type MockDaemonClient struct {
	sessions      map[string]*pb.SessionConfiguration
	executions    map[string]*MockExecution
	createErr     error
	executeErr    error
	statusErr     error
	cancelErr     error
	healthErr     error
}

// MockExecution represents a mock execution
type MockExecution struct {
	ExecutionID string
	SessionID   string
	Status      pb.ExecutionStatus
	Complete    bool
	ExitCode    int32
	Error       string
	StartedAt   time.Time
}

// NewMockDaemonClient creates a new mock daemon client
func NewMockDaemonClient() *MockDaemonClient {
	return &MockDaemonClient{
		sessions:   make(map[string]*pb.SessionConfiguration),
		executions: make(map[string]*MockExecution),
	}
}

// CreateSession creates a mock session
func (m *MockDaemonClient) CreateSession(ctx context.Context, sessionID string, config *pb.SessionConfiguration) (*pb.CreateSessionResponse, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	m.sessions[sessionID] = config
	return &pb.CreateSessionResponse{
		SessionId: sessionID,
		Status:    pb.SessionStatus_SESSION_STATUS_ACTIVE,
	}, nil
}

// GetSession gets a mock session
func (m *MockDaemonClient) GetSession(ctx context.Context, sessionID string) (*pb.GetSessionResponse, error) {
	config, ok := m.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}

	return &pb.GetSessionResponse{
		SessionId:     sessionID,
		Status:        pb.SessionStatus_SESSION_STATUS_ACTIVE,
		Configuration: config,
	}, nil
}

// DeleteSession deletes a mock session
func (m *MockDaemonClient) DeleteSession(ctx context.Context, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

// ExecuteSnippet executes a mock snippet
func (m *MockDaemonClient) ExecuteSnippet(ctx context.Context, sessionID string, executionID string, code string, language string, env map[string]string, args []string) (*pb.ExecuteSnippetResponse, error) {
	if m.executeErr != nil {
		return nil, m.executeErr
	}

	exec := &MockExecution{
		ExecutionID: executionID,
		SessionID:   sessionID,
		Status:      pb.ExecutionStatus_EXECUTION_STATUS_RUNNING,
		Complete:    false,
		StartedAt:   time.Now(),
	}

	m.executions[executionID] = exec

	return &pb.ExecuteSnippetResponse{
		ExecutionId: executionID,
		Status:      pb.ExecutionStatus_EXECUTION_STATUS_RUNNING,
	}, nil
}

// GetExecutionStatus gets mock execution status
func (m *MockDaemonClient) GetExecutionStatus(ctx context.Context, sessionID string, executionID string) (*pb.GetExecutionStatusResponse, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}

	exec, ok := m.executions[executionID]
	if !ok {
		return nil, errors.New("execution not found")
	}

	// Simulate completion after 1 second
	if !exec.Complete && time.Since(exec.StartedAt) > 1*time.Second {
		exec.Complete = true
		exec.Status = pb.ExecutionStatus_EXECUTION_STATUS_COMPLETED
		exec.ExitCode = 0
	}

	return &pb.GetExecutionStatusResponse{
		ExecutionId: executionID,
		Status:      exec.Status,
		Complete:    exec.Complete,
		ExitCode:    exec.ExitCode,
		Error:       exec.Error,
	}, nil
}

// CancelExecution cancels a mock execution
func (m *MockDaemonClient) CancelExecution(ctx context.Context, sessionID string, executionID string) error {
	if m.cancelErr != nil {
		return m.cancelErr
	}

	exec, ok := m.executions[executionID]
	if !ok {
		return errors.New("execution not found")
	}

	exec.Status = pb.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	exec.Complete = true
	return nil
}

// Health checks mock daemon health
func (m *MockDaemonClient) Health(ctx context.Context) error {
	return m.healthErr
}

// Close closes the mock client
func (m *MockDaemonClient) Close() error {
	return nil
}

// SetCreateSessionError sets an error for CreateSession
func (m *MockDaemonClient) SetCreateSessionError(err error) {
	m.createErr = err
}

// SetExecuteError sets an error for ExecuteSnippet
func (m *MockDaemonClient) SetExecuteError(err error) {
	m.executeErr = err
}

// SetStatusError sets an error for GetExecutionStatus
func (m *MockDaemonClient) SetStatusError(err error) {
	m.statusErr = err
}

// SetCancelError sets an error for CancelExecution
func (m *MockDaemonClient) SetCancelError(err error) {
	m.cancelErr = err
}

// SetHealthError sets an error for Health
func (m *MockDaemonClient) SetHealthError(err error) {
	m.healthErr = err
}

// CompleteExecution marks an execution as complete
func (m *MockDaemonClient) CompleteExecution(executionID string, exitCode int32) {
	if exec, ok := m.executions[executionID]; ok {
		exec.Complete = true
		exec.Status = pb.ExecutionStatus_EXECUTION_STATUS_COMPLETED
		exec.ExitCode = exitCode
	}
}

// FailExecution marks an execution as failed
func (m *MockDaemonClient) FailExecution(executionID string, errorMsg string) {
	if exec, ok := m.executions[executionID]; ok {
		exec.Complete = true
		exec.Status = pb.ExecutionStatus_EXECUTION_STATUS_FAILED
		exec.ExitCode = 1
		exec.Error = errorMsg
	}
}

// Ensure MockDaemonClient implements DaemonClient interface
var _ driver.DaemonClient = (*MockDaemonClient)(nil)

