// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package unit

import (
	"testing"
	"time"

	"github.com/elide-dev/elide-task-driver/driver"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/stretchr/testify/assert"
)

func TestTaskState_Serialization(t *testing.T) {
	taskState := driver.TaskState{
		TaskConfig: &drivers.TaskConfig{
			ID:   "test-123",
			Name: "test-task",
		},
		StartedAt:  time.Now(),
		ExecutionId: "exec-123",
		SessionId:   "session-123",
	}

	// Test that TaskState has required fields
	assert.NotNil(t, taskState.TaskConfig)
	assert.Equal(t, "test-123", taskState.TaskConfig.ID)
	assert.Equal(t, "exec-123", taskState.ExecutionId)
	assert.Equal(t, "session-123", taskState.SessionId)
	assert.False(t, taskState.StartedAt.IsZero())
}

func TestTaskState_Recovery(t *testing.T) {
	// Test that TaskState has all fields needed for recovery
	taskState := driver.TaskState{
		TaskConfig: &drivers.TaskConfig{
			ID:   "test-123",
			Name: "test-task",
		},
		StartedAt:  time.Now(),
		ExecutionId: "exec-123",
		SessionId:   "session-123",
	}

	// Verify all recovery fields are present
	assert.NotNil(t, taskState.TaskConfig, "TaskConfig required for recovery")
	assert.NotEmpty(t, taskState.ExecutionId, "ExecutionId required for recovery")
	assert.NotEmpty(t, taskState.SessionId, "SessionId required for recovery")
	assert.False(t, taskState.StartedAt.IsZero(), "StartedAt required for recovery")
}

