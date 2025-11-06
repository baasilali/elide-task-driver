// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"sync"
	"time"

	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/structs"
)

// TaskState is the runtime state which is encoded in the handle returned to
// Nomad client. This information is needed to rebuild the task state and
// handler during recovery.
type TaskState struct {
	ReattachConfig *structs.ReattachConfig
	TaskConfig     *drivers.TaskConfig
	StartedAt      time.Time

	// TODO: Once daemon API is available, add:
	// ExecutionId string  // Execution ID from Elide daemon (for recovery)
	// DaemonSocket string // Daemon socket path (in case it changed)
}

// taskStore provides a mechanism to store and retrieve task handles
// given a string identifier. The ID should be unique per task.
type taskStore struct {
	store map[string]*taskHandle
	lock  sync.RWMutex
}

func newTaskStore() *taskStore {
	return &taskStore{store: map[string]*taskHandle{}}
}

func (ts *taskStore) Set(id string, handle *taskHandle) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.store[id] = handle
}

func (ts *taskStore) Get(id string) (*taskHandle, bool) {
	ts.lock.RLock()
	defer ts.lock.RUnlock()
	t, ok := ts.store[id]
	return t, ok
}

func (ts *taskStore) Delete(id string) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	delete(ts.store, id)
}

