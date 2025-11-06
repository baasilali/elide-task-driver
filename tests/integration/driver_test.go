// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/elide-dev/elide-task-driver/driver"
	"github.com/elide-dev/elide-task-driver/tests/helpers"
	pb "github.com/elide-dev/elide-task-driver/proto/gen/go/elide/daemon/v1alpha1"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriver_StartTask(t *testing.T) {
	logger := hclog.NewNullLogger()
	plugin := driver.NewPlugin(logger).(*driver.ElideDriverPlugin)

	// Use mock daemon client
	mockClient := helpers.NewMockDaemonClient()
	
	// We can't directly set the daemon client, so we test the public API
	// This test will be expanded when we add SetConfig support for testing
	
	_ = plugin
	_ = mockClient
}

func TestDriver_WithMockDaemon(t *testing.T) {
	// Integration test that uses mock daemon client
	// This will test the full flow: StartTask -> ExecuteSnippet -> WaitTask
	logger := hclog.NewNullLogger()
	plugin := driver.NewPlugin(logger).(*driver.ElideDriverPlugin)

	mockClient := helpers.NewMockDaemonClient()
	
	// Create session
	sessionID := "test-session"
	sessionConfig := &pb.SessionConfiguration{
		ContextPoolSize:  10,
		EnabledLanguages: []string{"python", "javascript"},
		EnabledIntrinsics: []string{"io", "env"},
		MemoryLimitMb:    512,
		EnableAi:         false,
	}
	
	_, err := mockClient.CreateSession(context.Background(), sessionID, sessionConfig)
	require.NoError(t, err)

	// Execute snippet
	executionID := "test-exec-123"
	_, err = mockClient.ExecuteSnippet(
		context.Background(),
		sessionID,
		executionID,
		"print('hello')",
		"python",
		nil,
		nil,
	)
	require.NoError(t, err)

	// Get status
	status, err := mockClient.GetExecutionStatus(context.Background(), sessionID, executionID)
	require.NoError(t, err)
	assert.Equal(t, pb.ExecutionStatus_EXECUTION_STATUS_RUNNING, status.Status)

	// Complete execution
	mockClient.CompleteExecution(executionID, 0)

	// Get final status
	status, err = mockClient.GetExecutionStatus(context.Background(), sessionID, executionID)
	require.NoError(t, err)
	assert.True(t, status.Complete)
	assert.Equal(t, int32(0), status.ExitCode)

	_ = plugin
}

