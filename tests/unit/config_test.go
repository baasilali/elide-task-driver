// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package unit

import (
	"testing"

	"github.com/elide-dev/elide-task-driver/driver"
	"github.com/stretchr/testify/assert"
)

func TestTaskConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  driver.TaskConfig
		wantErr bool
	}{
		{
			name: "valid with script",
			config: driver.TaskConfig{
				Script:   "test.py",
				Language: "python",
			},
			wantErr: false,
		},
		{
			name: "valid with code",
			config: driver.TaskConfig{
				Code:     "print('hello')",
				Language: "python",
			},
			wantErr: false,
		},
		{
			name: "invalid - no script or code",
			config: driver.TaskConfig{
				Language: "python",
			},
			wantErr: true,
		},
		{
			name: "invalid - both script and code",
			config: driver.TaskConfig{
				Script:   "test.py",
				Code:     "print('hello')",
				Language: "python",
			},
			wantErr: true,
		},
		{
			name: "invalid - unsupported language",
			config: driver.TaskConfig{
				Script:   "test.rb",
				Language: "ruby",
			},
			wantErr: true,
		},
		{
			name: "valid - python",
			config: driver.TaskConfig{
				Script:   "test.py",
				Language: "python",
			},
			wantErr: false,
		},
		{
			name: "valid - javascript",
			config: driver.TaskConfig{
				Script:   "test.js",
				Language: "javascript",
			},
			wantErr: false,
		},
		{
			name: "valid - typescript",
			config: driver.TaskConfig{
				Script:   "test.ts",
				Language: "typescript",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSessionConfig_Defaults(t *testing.T) {
	config := driver.SessionConfig{}
	
	// Test that defaults are reasonable
	assert.Equal(t, 0, config.ContextPoolSize)
	assert.Equal(t, 0, len(config.EnabledLanguages))
	assert.Equal(t, 0, len(config.EnabledIntrinsics))
	assert.Equal(t, 0, config.MemoryLimitMB)
	assert.Equal(t, false, config.EnableAI)
}

func TestElideOptions(t *testing.T) {
	opts := driver.ElideOptions{
		MemoryLimit: 256,
		EnableAI:    true,
		Timeout:     30,
	}

	assert.Equal(t, 256, opts.MemoryLimit)
	assert.Equal(t, true, opts.EnableAI)
	assert.Equal(t, 30, opts.Timeout)
}

