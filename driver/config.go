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
		// Path to script file (relative to task directory) - optional since 'code' can be used instead
		"script": hclspec.NewAttr("script", "string", false),
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
		// Elide-specific options (reserved for future use)
		// NOTE: These are currently defined but NOT USED. They are reserved for when
		// the daemon API supports per-task configuration overrides. Currently, all
		// tasks use the session-level configuration defined in the driver config.
		// See API_QUESTIONS.md for details on when this will be supported.
		"elide_opts": hclspec.NewBlock("elide_opts", false, hclspec.NewObject(map[string]*hclspec.Spec{
			// Memory limit in MB (per-task override - not yet supported)
			"memory_limit": hclspec.NewAttr("memory_limit", "number", false),
			// Enable AI features (per-task override - not yet supported)
			"enable_ai": hclspec.NewAttr("enable_ai", "bool", false),
			// Timeout in seconds (not yet supported by daemon API)
			"timeout": hclspec.NewAttr("timeout", "number", false),
		})),
	})
)

// Config is the driver configuration set by the Nomad agent
type Config struct {
	ElideBinary   string        `codec:"elide_binary"`
	DaemonSocket  string        `codec:"daemon_socket"`
	DaemonAddress string        `codec:"daemon_address"`
	SessionConfig SessionConfig `codec:"session_config"`
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

// ElideOptions contains Elide-specific per-task configuration
// NOTE: These fields are currently RESERVED FOR FUTURE USE and are not applied.
// All tasks currently use session-level configuration from the driver config.
// These will be used when the daemon API supports per-task overrides.
type ElideOptions struct {
	MemoryLimit int  `codec:"memory_limit"` // Per-task memory limit (not yet supported)
	EnableAI    bool `codec:"enable_ai"`    // Per-task AI enable (not yet supported)
	Timeout     int  `codec:"timeout"`      // Execution timeout in seconds (not yet supported)
}

// Validate checks if the task configuration is valid
func (tc *TaskConfig) Validate() error {
	if tc.Script == "" && tc.Code == "" {
		return fmt.Errorf("either 'script' or 'code' must be specified")
	}
	if tc.Script != "" && tc.Code != "" {
		return fmt.Errorf("cannot specify both 'script' and 'code'")
	}
	// Basic language validation - actual validation against session config happens in driver
	if tc.Language == "" {
		return fmt.Errorf("language must be specified")
	}
	return nil
}

// ValidateLanguage checks if the requested language is enabled in the session configuration
func (tc *TaskConfig) ValidateLanguage(enabledLanguages []string) error {
	for _, lang := range enabledLanguages {
		if tc.Language == lang {
			return nil
		}
	}
	return fmt.Errorf("language %q not enabled in session (enabled: %v)", tc.Language, enabledLanguages)
}


