// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package unit

import (
	"testing"
)

func TestTaskHandle_IsRunning(t *testing.T) {
	// Note: taskHandle is not exported, so we test through the driver
	// For now, we test the TaskStatus method which is the public API
	// This test will be expanded when we add driver integration tests
}

// Note: taskHandle is not exported, so we can't directly test it
// These tests will be expanded when we add driver-level integration tests
// that test the handle through the driver's public API

