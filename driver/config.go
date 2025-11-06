// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"fmt"

	"github.com/hashicorp/nomad/plugins/shared/hclspec"
)

var (
	// configSpec is the HCL specification for the driver plugin configuration
	// This is set at the Nomad agent level (plugin stanza)
	configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		// Path to Elide daemon binary (if driver should start it)
		"elide_binary": hclspec.NewDefault(
			hclspec.NewAttr("elide_binary", "string", false),
			hclspec.NewLiteral(`"/usr/local/bin/elide"`),
		),
		// Unix socket path for Elide daemon (if daemon is pre-started)
		"daemon_socket": hclspec.NewDefault(
			hclspec.NewAttr("daemon_socket", "string", false),
			hclspec.NewLiteral(`"/tmp/elide-daemon.sock"`),
		),
		// TCP address for Elide daemon (alternative to Unix socket)
		"daemon_address": hclspec.NewAttr("daemon_address", "string", false),
		// Whether driver should start daemon if not running
		"auto_start_daemon": hclspec.NewDefault(
			hclspec.NewAttr("auto_start_daemon", "bool", false),
			hclspec.NewLiteral("true"),
		),
		// Session configuration (one per Nomad client)
		"session_config": hclspec.NewBlock("session_config", false, hclspec.NewObject(map[string]*hclspec.Spec{
			"context_pool_size": hclspec.NewDefault(
				hclspec.NewAttr("context_pool_size", "number", false),
				hclspec.NewLiteral("10"),
			),
			"enabled_languages": hclspec.NewDefault(
				hclspec.NewAttr("enabled_languages", "list(string)", false),
				hclspec.NewLiteral(`["python", "javascript", "typescript"]`),
			),
			"enabled_intrinsics": hclspec.NewDefault(
				hclspec.NewAttr("enabled_intrinsics", "list(string)", false),
				hclspec.NewLiteral(`["io", "env"]`),
			),
			"memory_limit_mb": hclspec.NewDefault(
				hclspec.NewAttr("memory_limit_mb", "number", false),
				hclspec.NewLiteral("512"),
			),
			"enable_ai": hclspec.NewDefault(
				hclspec.NewAttr("enable_ai", "bool", false),
				hclspec.NewLiteral("false"),
			),
		})),
	})

	// taskConfigSpec is the HCL specification for task configuration
	// This is set per-task in the job spec
	taskConfigSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		// Path to script file (relative to task directory)
		"script": hclspec.NewAttr("script", "string", true),
		// Inline code (alternative to script file)
		"code": hclspec.NewAttr("code", "string", false),
		// Language: "python", "javascript", "typescript"
		"language": hclspec.NewDefault(
			hclspec.NewAttr("language", "string", false),
			hclspec.NewLiteral(`"python"`),
		),
		// Arguments to pass to script
		"args": hclspec.NewAttr("args", "list(string)", false),
		// Environment variables
		"env": hclspec.NewAttr("env", "map(string)", false),
		// Elide-specific options
		"elide_opts": hclspec.NewBlock("elide_opts", false, hclspec.NewObject(map[string]*hclspec.Spec{
			// Memory limit in MB
			"memory_limit": hclspec.NewAttr("memory_limit", "number", false),
			// Enable AI features
			"enable_ai": hclspec.NewAttr("enable_ai", "bool", false),
			// Timeout in seconds
			"timeout": hclspec.NewAttr("timeout", "number", false),
		})),
	})
)

// Config is the driver configuration set by the Nomad agent
type Config struct {
	ElideBinary     string            `codec:"elide_binary"`
	DaemonSocket    string            `codec:"daemon_socket"`
	DaemonAddress   string            `codec:"daemon_address"`
	AutoStartDaemon bool              `codec:"auto_start_daemon"`
	SessionConfig   SessionConfig     `codec:"session_config"`
}

// SessionConfig is the session configuration
type SessionConfig struct {
	ContextPoolSize  int      `codec:"context_pool_size"`
	EnabledLanguages []string `codec:"enabled_languages"`
	EnabledIntrinsics []string `codec:"enabled_intrinsics"`
	MemoryLimitMB    int      `codec:"memory_limit_mb"`
	EnableAI         bool     `codec:"enable_ai"`
}

// TaskConfig is the per-task configuration
type TaskConfig struct {
	// Script path (relative to task directory)
	Script string `codec:"script"`
	// Inline code (alternative to script)
	Code string `codec:"code"`
	// Language: python, javascript, typescript
	Language string `codec:"language"`
	// Arguments to pass to script
	Args []string `codec:"args"`
	// Environment variables
	Env map[string]string `codec:"env"`
	// Elide-specific options
	ElideOpts ElideOptions `codec:"elide_opts"`
}

// ElideOptions contains Elide-specific configuration
type ElideOptions struct {
	MemoryLimit int  `codec:"memory_limit"`
	EnableAI    bool `codec:"enable_ai"`
	Timeout     int  `codec:"timeout"` // seconds
}

// Validate checks if the task configuration is valid
func (tc *TaskConfig) Validate() error {
	if tc.Script == "" && tc.Code == "" {
		return fmt.Errorf("either 'script' or 'code' must be specified")
	}
	if tc.Script != "" && tc.Code != "" {
		return fmt.Errorf("cannot specify both 'script' and 'code'")
	}
	if tc.Language != "python" && tc.Language != "javascript" && tc.Language != "typescript" {
		return fmt.Errorf("unsupported language: %s (supported: python, javascript, typescript)", tc.Language)
	}
	return nil
}

