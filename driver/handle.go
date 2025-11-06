// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins/drivers"
)

// taskHandle stores runtime information for a running task
type taskHandle struct {
	// stateLock syncs access to all fields below
	stateLock sync.RWMutex

	logger      hclog.Logger
	taskConfig  *drivers.TaskConfig
	startedAt   time.Time
	completedAt time.Time
	exitResult  *drivers.ExitResult

	// TODO: Once daemon API is available, add:
	// executionId string  // Execution ID from Elide daemon
	// status      string  // Current execution status (running, completed, failed)
	// daemonClient DaemonClient  // Reference to daemon client (if needed per-task)
}

// TaskStatus returns the current status of the task
func (h *taskHandle) TaskStatus() *drivers.TaskStatus {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()

	state := drivers.TaskStateUnknown
	if h.exitResult == nil {
		state = drivers.TaskStateRunning
	} else if h.exitResult.ExitCode == 0 {
		state = drivers.TaskStateExited
	} else {
		state = drivers.TaskStateExited
	}

	return &drivers.TaskStatus{
		ID:          h.taskConfig.ID,
		Name:        h.taskConfig.Name,
		State:       state,
		StartedAt:   h.startedAt,
		CompletedAt: h.completedAt,
		ExitResult:  h.exitResult,
		DriverAttributes: map[string]string{
			// TODO: Add execution ID once daemon API is available
			// "execution_id": h.executionId,
		},
	}
}

// IsRunning returns whether the task is currently running
func (h *taskHandle) IsRunning() bool {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()
	return h.exitResult == nil
}

// SetCompleted marks the task as completed with the given exit result
func (h *taskHandle) SetCompleted(result *drivers.ExitResult) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	h.exitResult = result
	h.completedAt = time.Now()
}

// TODO: Once daemon API is available, add methods for:
// - Polling execution status from daemon
// - Updating status based on daemon responses
// - Handling cancellation

