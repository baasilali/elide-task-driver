# Elide Task Driver for Nomad

> **A HashiCorp Nomad task driver plugin that runs multiple Elide code snippets in a single shared daemon instance for maximum resource efficiency**

[![Elide](https://img.shields.io/badge/Elide-v1.0--beta-blue)](https://elide.dev)
[![Nomad](https://img.shields.io/badge/Nomad-1.9+-purple)](https://www.nomadproject.io)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://golang.org)
[![Protocol Buffers](https://img.shields.io/badge/Protocol_Buffers-gRPC-green)](https://grpc.io)

---

## Table of Contents

- [Overview](#overview)
- [What is This?](#what-is-this)
- [Architecture](#architecture)
- [Key Features](#key-features)
- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [Implementation Guide](#implementation-guide)
- [Communication Strategy](#communication-strategy)
- [Testing](#testing)
- [Example Job Specs](#example-job-specs)
- [API Integration](#api-integration)
- [Success Criteria](#success-criteria)
- [Resources](#resources)
- [Contributing](#contributing)

---

## ✅ Current Status

**Status**: ✅ **Driver is fully functional and working end-to-end!**

**Architecture**: Implemented **session-based API** with **one Elide daemon per Nomad client** running multiple code snippets in isolated sessions.

**Implementation Status**:
- ✅ **Session-based API** implemented (per Dario's feedback)
- ✅ **gRPC proto spec** drafted with session management
- ✅ **Stubbed gRPC server** created for testing
- ✅ **Driver fully implemented** with session support
- ✅ **End-to-end testing** successful with stubbed server
- ✅ **Ready for real Elide daemon** (just swap in the real server)

**Key Features**:
- **Session Isolation**: One session per Nomad client with isolated context pools
- **Minimal Intrinsics**: Configurable intrinsics for sandbox guarantees
- **gRPC Integration**: Full gRPC communication via Protocol Buffers
- **Stubbed Server**: Allows full development without waiting for Elide feature

**What Works**:
- ✅ Plugin loads in Nomad
- ✅ Session creation on driver initialization
- ✅ Task execution within sessions
- ✅ Execution status polling
- ✅ Task cancellation
- ✅ Task recovery after restart

---

## Overview

The **Elide Task Driver** is a custom Nomad plugin written in Go that enables HashiCorp Nomad to orchestrate and manage Elide runtime instances. Instead of running containerized workloads or virtual machines, this driver executes polyglot applications directly on the Elide runtime with native performance and seamless multi-language interoperability.

### Key Innovation: One Daemon, Multiple Snippets

**Unlike traditional approaches**, this driver uses **one Elide daemon per Nomad client node** that executes multiple code snippets (tasks) in a shared GraalVM context. This provides:

- **Resource Efficiency**: One GraalVM instance vs many (massive memory savings)
- **Performance**: Shared context initialization (faster execution)
- **Scalability**: More tasks per node (higher density)
- **Cost Savings**: Lower resource usage (better utilization)

### What Makes This Special?

1. **Resource Efficient**: One Elide daemon per Nomad client, multiple snippets in shared context
2. **Native Performance**: No container overhead - run polyglot apps directly
3. **Multi-Language**: Execute Python, JavaScript, TypeScript, and JVM languages in one process
4. **Built-in AI**: Leverage Elide's local AI inference without external services
5. **Lightweight**: Minimal resource footprint compared to traditional container orchestration
6. **gRPC Integration**: Type-safe communication via Protocol Buffers (when available)
7. **Nomad Native**: Full integration with Nomad's scheduling, monitoring, and lifecycle management

---

## What is This?

### The Problem
You want to run Elide applications on a Nomad cluster, but Nomad doesn't natively understand how to execute Elide workloads. You could wrap Elide in containers, but that adds overhead and complexity.

### The Solution
A **task driver plugin** that acts as a bridge, using **one Elide daemon per Nomad client** to run multiple code snippets:

```
Nomad Scheduler
    ↓ (HCL job specification)
Elide Task Driver (Go)
    ↓ (gRPC/API - when available)
Elide Daemon (Single GraalVM Instance)
    ├── Task 1 → Snippet 1 (isolated execution)
    ├── Task 2 → Snippet 2 (isolated execution)
    └── Task 3 → Snippet 3 (isolated execution)
    ↑ (output, logs, metrics)
Back to Nomad
```

**Architecture Benefits**:
- **One daemon** per Nomad client node (not one per task)
- **Shared GraalVM context** for efficiency
- **Multiple snippets** executed concurrently
- **Resource efficient** compared to spawning processes per task

### Comparison

| Approach | Overhead | Complexity | Elide Features | Resource Efficiency |
|----------|----------|------------|----------------|---------------------|
| **Docker + Elide** | High (container runtime) | Medium | All | ❌ Multiple containers |
| **Raw Exec Driver** | Low | Low | No lifecycle management | ❌ Multiple processes |
| **Elide Task Driver** | Minimal | Low | All + Nomad integration | ✅ **One daemon, multiple snippets** |

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Nomad Control Plane                       │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Nomad Server (Scheduler)                               │ │
│  │ - Job placement                                        │ │
│  │ - Resource allocation                                  │ │
│  │ - Health monitoring                                    │ │
│  └────────────┬───────────────────────────────────────────┘ │
└───────────────┼─────────────────────────────────────────────┘
                │ Plugin Protocol (gRPC)
                ↓
┌─────────────────────────────────────────────────────────────┐
│                    Nomad Client Node                         │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Nomad Agent                                            │ │
│  │ ┌──────────────────────────────────────────────────┐   │ │
│  │ │ Plugin Manager                                   │   │ │
│  │ │ - Loads task driver plugins                      │   │ │
│  │ │ - Manages plugin lifecycle                       │   │ │
│  │ └─────────────┬────────────────────────────────────┘   │ │
│  └───────────────┼────────────────────────────────────────┘ │
│                  │ go-plugin (HashiCorp)                    │
│                  ↓                                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Elide Task Driver Plugin (This Project)                │ │
│  │ ┌──────────────────────────────────────────────────┐   │ │
│  │ │ Core Driver Logic                                │   │ │
│  │ │ - TaskConfigSchema()                             │   │ │
│  │ │ - StartTask()                                    │   │ │
│  │ │ - StopTask()                                     │   │ │
│  │ │ - InspectTask()                                  │   │ │
│  │ └──────────────────────────────────────────────────┘   │ │
│  │ ┌──────────────────────────────────────────────────┐   │ │
│  │ │ Task Handle Management                           │   │ │
│  │ │ - Process lifecycle                              │   │ │
│  │ │ - Resource tracking                              │   │ │
│  │ │ - Event publishing                               │   │ │
│  │ └──────────────────────────────────────────────────┘   │ │
│  │ ┌──────────────────────────────────────────────────┐   │ │
│  │ │ Elide API Client (gRPC/HTTP)                       │   │ │
│  │ │ - Proto-generated stubs                          │   │ │
│  │ │ - Unix socket or TCP transport                   │   │ │
│  │ └─────────────┬────────────────────────────────────┘   │ │
│  └───────────────┼────────────────────────────────────────┘ │
│                  │ Unix Socket / TCP / API                   │
│                  ↓                                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Elide Daemon (Single Instance per Nomad Client)        │ │
│  │ ┌──────────────────────────────────────────────────┐   │ │
│  │ │ GraalVM Polyglot Engine (Shared Context)         │   │ │
│  │ │ ┌────────────────┐  ┌────────────────┐          │   │ │
│  │ │ │ Python VM      │  │ JavaScript VM  │          │   │ │
│  │ │ └────────────────┘  └────────────────┘          │   │ │
│  │ └──────────────────────────────────────────────────┘   │ │
│  │ ┌──────────────────────────────────────────────────┐   │ │
│  │ │ Elide Intrinsics (Shared)                        │   │ │
│  │ │ - Local AI (llama.cpp)                           │   │ │
│  │ │ - HTTP Server                                    │   │ │
│  │ │ - SQLite                                         │   │ │
│  │ └──────────────────────────────────────────────────┘   │ │
│  │ ┌──────────────────────────────────────────────────┐   │ │
│  │ │ Multiple Task Snippets (Isolated Execution)     │   │ │
│  │ │ ├── Task 1: Python script                        │   │ │
│  │ │ ├── Task 2: JavaScript snippet                  │   │ │
│  │ │ ├── Task 3: TypeScript code                     │   │ │
│  │ │ └── Task 4: Kotlin/Java code                    │   │ │
│  │ └──────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```mermaid
sequenceDiagram
    participant User
    participant Nomad
    participant Driver
    participant ElideDaemon
    participant Snippet

    Note over Driver,ElideDaemon: One Elide Daemon per Nomad Client

    User->>Nomad: Submit job (HCL)
    Nomad->>Nomad: Schedule task
    Nomad->>Driver: StartTask(config)
    Driver->>Driver: Parse task config
    
    alt Daemon not running
        Driver->>ElideDaemon: Start daemon (once per client)
        ElideDaemon-->>Driver: Daemon ready
    end
    
    Driver->>ElideDaemon: ExecuteSnippet(code, env, args)
    ElideDaemon->>Snippet: Execute in shared context
    Snippet-->>ElideDaemon: Result (stdout, stderr, exit_code)
    ElideDaemon-->>Driver: Snippet result
    Driver->>Driver: Create task handle
    Driver-->>Nomad: Task started
    
    loop Health Checks
        Nomad->>Driver: InspectTask()
        Driver->>ElideDaemon: GetSnippetStatus(task_id)
        ElideDaemon-->>Driver: Status (running, complete, failed)
        Driver-->>Nomad: Task healthy
    end
    
    User->>Nomad: Stop job
    Nomad->>Driver: StopTask()
    Driver->>ElideDaemon: CancelSnippet(task_id)
    ElideDaemon->>Snippet: Cancel execution
    Snippet-->>ElideDaemon: Cancelled
    ElideDaemon-->>Driver: Snippet stopped
    Driver-->>Nomad: Task stopped
```

---

## Key Features

### Planned Features

#### **MVP (Minimum Viable Product)**
- [x] Driver plugin skeleton
- [x] Architecture design (one daemon, multiple snippets)
- [ ] **Feature request to Elide team** (daemon mode)
- [ ] Load in Nomad without errors
- [ ] Start Elide daemon (one per Nomad client)
- [ ] Execute snippets in daemon (Python, JS, TS)
- [ ] Capture stdout/stderr for logs
- [ ] Report task completion/failure
- [ ] Stop running tasks (cancel snippet)
- [ ] Basic resource limits
- [ ] **Temporary workaround** (if daemon mode not available yet)

#### **V1.0** (After Elide Daemon Mode Available)
- [ ] gRPC daemon mode integration
- [ ] ExecuteSnippet API implementation
- [ ] Resource monitoring (CPU, memory) per daemon
- [ ] Multi-language support configuration
- [ ] Health checks via Elide APIs
- [ ] Graceful shutdown handling
- [ ] Task recovery after Nomad agent restart
- [ ] Environment variable injection
- [ ] Artifact download support
- [ ] Concurrent snippet execution tracking

#### **V2.0 (Future)**
- [ ] AI workload-specific optimizations
- [ ] Hot reload support
- [ ] Metrics export (Prometheus format)
- [ ] Integration with Nomad service mesh
- [ ] Volume mounting
- [ ] GPU support (if Elide adds it)

---

## Prerequisites

### Required Software

#### **1. Go Development Environment**
```bash
# Install Go 1.21 or later
brew install go        # macOS
# or
sudo apt install golang-go  # Ubuntu/Debian

# Verify installation
go version  # Should show 1.21+
```

#### **2. Nomad**
```bash
# Install Nomad 1.9+
brew install nomad     # macOS
# or download from https://www.nomadproject.io/downloads

# Verify installation
nomad version
```

#### **3. Protocol Buffers & gRPC Tools**
```bash
# Install protoc compiler
brew install protobuf

# Install Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Add to PATH (add to ~/.zshrc or ~/.bashrc)
export PATH="$PATH:$(go env GOPATH)/bin"
```

#### **4. Buf CLI** (for proto management)
```bash
# Install buf
brew install bufbuild/buf/buf

# Verify installation
buf --version
```

#### **5. Elide Runtime**
```bash
# Already installed based on your workspace
elide --version  # Should show v1.0.0-beta9+
```

### Optional Tools

- **Docker**: For comparison testing
- **jq**: For JSON parsing in tests
- **make**: For build automation

---

## Project Structure

```
elide-task-driver/
│
├── README.md                   # This file
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
├── Makefile                    # Build automation
├── LICENSE                     # Apache 2.0 license
│
├── main.go                     # Plugin entry point
│   └── Initializes and serves the driver plugin
│
├── driver/                     # Core driver implementation
│   ├── driver.go              # Main driver struct and interface
│   ├── config.go              # Configuration parsing and validation
│   ├── handle.go              # Task handle (running task state)
│   ├── elide.go               # Elide-specific execution logic
│   ├── stats.go               # Resource usage collection
│   └── fingerprint.go         # Driver capability detection
│
├── proto/                      # Protocol Buffer definitions
│   ├── buf.yaml               # Buf configuration
│   ├── buf.gen.yaml           # Code generation config
│   ├── buf.lock               # Dependency lock file
│   └── gen/                   # Generated Go code (gitignored)
│       └── go/
│           └── elide/
│               ├── call/
│               ├── control/
│               └── tools/
│
├── examples/                   # Example Nomad job specifications
│   ├── hello-python.nomad     # Simple Python script
│   ├── hello-javascript.nomad # Simple JS script
│   ├── http-server.nomad      # Web server example
│   ├── ai-inference.nomad     # Local AI workload
│   └── polyglot.nomad         # Multi-language app
│
├── tests/                      # Test suite
│   ├── driver_test.go         # Unit tests for driver
│   ├── integration_test.go    # Integration tests with Elide
│   ├── e2e_test.go            # End-to-end tests with Nomad
│   └── fixtures/              # Test fixtures (scripts, configs)
│
├── scripts/                    # Build and development scripts
│   ├── build.sh               # Build the plugin
│   ├── install.sh             # Install to Nomad plugin directory
│   ├── test-local.sh          # Run local tests
│   └── gen-proto.sh           # Generate proto code
│
└── docs/                       # Additional documentation
    ├── ARCHITECTURE.md        # Detailed architecture docs
    ├── API.md                 # API usage examples
    └── DEVELOPMENT.md         # Development guide
```

---

## Quick Start

### Prerequisites

- **Go 1.21+** - For building the driver
- **Nomad 1.9+** - For running the driver
- **Buf CLI** - For generating proto code (optional, already generated)
- **Make** - For build automation

### Step 1: Build the Driver

```bash
cd /path/to/elide-task-driver

# Build the driver plugin
make build

# Verify plugin was created
ls -lh build/plugins/elide-task-driver
```

### Step 2: Create Plugin Symlink

Nomad discovers plugins by name, so we need a symlink:

```bash
# Create symlink so Nomad can find the plugin by name "elide"
cd build/plugins
ln -sf elide-task-driver elide
ls -lh | grep elide
```

### Step 3: Update Configuration

Edit `nomad-agent.hcl` to set the correct plugin directory path:

```hcl
plugin_dir = "/path/to/elide-task-driver/build/plugins"
```

### Step 4: Start Stubbed Server (Terminal 1)

The driver uses a stubbed gRPC server for testing:

```bash
cd /path/to/elide-task-driver
make server
```

You should see:
```
2025/11/05 20:12:46 Stubbed Elide daemon server listening on /tmp/elide-daemon.sock
```

### Step 5: Start Nomad with Driver (Terminal 2)

```bash
cd /path/to/elide-task-driver

# Start Nomad with the driver configuration
nomad agent -dev -config=nomad-agent.hcl
```

You should see:
```
[INFO]  agent: detected plugin: name=elide type=driver plugin_version=v0.1.0
[INFO]  client.driver_mgr.elide: created session: session_id=nomad-client-session
[DEBUG] client.driver_mgr: detected drivers: drivers="map[healthy:[raw_exec elide java]...]"
```

### Step 6: Verify Driver Loaded (Terminal 3)

```bash
# Check that the driver is available
nomad node status -self | grep -i driver
```

Should show: `Driver Status = elide,java,raw_exec`

### Step 7: Submit Test Job (Terminal 4)

```bash
cd /path/to/elide-task-driver

# Submit the example job
nomad job run examples/hello-python.nomad
```

You should see:
```
==> Allocation "ebe4e0f4" created: node "2ec1bfb8", group "python"
==> Evaluation "7b9501fc" finished with status "complete"
```

### Step 8: Verify Job Completed

```bash
# Check job status
nomad job status hello-python

# Check allocation status (use allocation ID from output above)
nomad alloc status <allocation-id>

# View logs
nomad alloc logs <allocation-id>
```

Expected output:
- **Status**: `complete`
- **Exit Code**: `0`
- **Duration**: ~2 seconds

---

## Session-Based Architecture

### Design Decision (Per CTO Feedback)

The driver implements a **session-based API** for better isolation and sandboxing:

- **One Session per Nomad Client**: Each Nomad client node gets its own isolated session
- **Isolated Context Pools**: Sessions prevent shared state between different callers
- **Minimal Intrinsics**: Configurable intrinsics for sandbox guarantees
- **Customizable Runtime**: Each session can have different runtime configurations

### Benefits

1. **Isolation**: Prevents shared state issues between different callers
2. **Sandboxing**: Additional buffer for sandboxing with isolated context pools
3. **Customization**: Different API clients can have different session configs
4. **Testing**: Stubbed server allows full development without waiting for Elide feature

### Session Configuration

Sessions are configured in `nomad-agent.hcl`:

```hcl
session_config {
  context_pool_size  = 10                    # Isolated pool per session
  enabled_languages  = ["python", "javascript", "typescript"]
  enabled_intrinsics = ["io", "env"]          # Minimal set for sandbox
  memory_limit_mb    = 512
  enable_ai          = false
}
```

See `DARIO_FEEDBACK.md` and `SESSION_API.md` for more details.

---

## Implementation Guide

### Phase 1: Core Driver Interface (Week 1-2)

#### Key Methods to Implement

##### **1. StartTask** - Most Important Method

```go
// StartTask launches a new Elide runtime instance
func (d *ElideDriver) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
    d.logger.Info("starting task", "task_id", cfg.ID)
    
    // 1. Decode task configuration
    var taskConfig TaskConfig
    if err := cfg.DecodeDriverConfig(&taskConfig); err != nil {
        return nil, nil, fmt.Errorf("failed to decode config: %v", err)
    }
    
    // 2. Validate configuration
    if err := taskConfig.Validate(); err != nil {
        return nil, nil, fmt.Errorf("invalid config: %v", err)
    }
    
    // 3. Prepare execution environment
    elideBinary := taskConfig.ElideOpts.ElideBinary
    if elideBinary == "" {
        elideBinary = "/usr/local/bin/elide"
    }
    
    // 4. Build command
    args := []string{"run"}
    
    // Add language flags if specified
    for _, lang := range taskConfig.ElideOpts.Languages {
        args = append(args, "--language", lang)
    }
    
    // Add the script path
    args = append(args, taskConfig.Script)
    
    // Add script arguments
    args = append(args, taskConfig.Args...)
    
    cmd := exec.Command(elideBinary, args...)
    
    // 5. Set up environment
    cmd.Env = d.buildEnv(cfg.Env, taskConfig.Env)
    cmd.Dir = cfg.TaskDir().Dir
    
    // 6. Set up stdio for logging
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get stdout: %v", err)
    }
    
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get stderr: %v", err)
    }
    
    // 7. Start the process
    if err := cmd.Start(); err != nil {
        return nil, nil, fmt.Errorf("failed to start elide: %v", err)
    }
    
    d.logger.Info("elide process started", "pid", cmd.Process.Pid)
    
    // 8. Create task handle
    handle := &TaskHandle{
        taskID:    cfg.ID,
        pid:       cmd.Process.Pid,
        command:   cmd,
        startedAt: time.Now(),
        logger:    d.logger.With("task_id", cfg.ID),
    }
    
    // 9. Start log collection goroutines
    go handle.collectLogs("stdout", stdout)
    go handle.collectLogs("stderr", stderr)
    
    // 10. Monitor process exit
    go handle.monitorExit()
    
    // 11. Store handle
    d.tasks[cfg.ID] = handle
    
    // 12. Return handle to Nomad
    driverHandle := &drivers.TaskHandle{
        Version: 1,
        Config:  cfg,
    }
    
    return driverHandle, nil, nil
}
```

##### **2. StopTask** - Graceful Shutdown

```go
func (d *ElideDriver) StopTask(taskID string, timeout time.Duration, signal string) error {
    d.logger.Info("stopping task", "task_id", taskID)
    
    handle, ok := d.tasks[taskID]
    if !ok {
        return fmt.Errorf("task not found: %s", taskID)
    }
    
    // Send signal to process
    sig := syscall.SIGTERM
    if signal != "" {
        // Parse signal string if provided
        sig = parseSignal(signal)
    }
    
    if err := handle.command.Process.Signal(sig); err != nil {
        return fmt.Errorf("failed to signal process: %v", err)
    }
    
    // Wait for graceful shutdown with timeout
    done := make(chan error, 1)
    go func() {
        done <- handle.command.Wait()
    }()
    
    select {
    case <-time.After(timeout):
        // Timeout - force kill
        d.logger.Warn("task did not stop gracefully, killing", "task_id", taskID)
        handle.command.Process.Kill()
        return fmt.Errorf("task did not stop within timeout")
    case err := <-done:
        d.logger.Info("task stopped", "task_id", taskID, "error", err)
        return nil
    }
}
```

##### **3. InspectTask** - Status Reporting

```go
func (d *ElideDriver) InspectTask(taskID string) (*drivers.TaskStatus, error) {
    handle, ok := d.tasks[taskID]
    if !ok {
        return nil, drivers.ErrTaskNotFound
    }
    
    status := &drivers.TaskStatus{
        ID:          taskID,
        Name:        handle.taskName,
        State:       handle.getState(),
        StartedAt:   handle.startedAt,
        CompletedAt: handle.completedAt,
        ExitResult:  handle.exitResult,
        DriverAttributes: map[string]string{
            "pid":     fmt.Sprintf("%d", handle.pid),
            "version": pluginVersion,
        },
    }
    
    return status, nil
}
```

##### **4. TaskStats** - Resource Monitoring

```go
func (d *ElideDriver) TaskStats(ctx context.Context, taskID string, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
    handle, ok := d.tasks[taskID]
    if !ok {
        return nil, drivers.ErrTaskNotFound
    }
    
    ch := make(chan *drivers.TaskResourceUsage)
    go handle.emitStats(ctx, ch, interval)
    
    return ch, nil
}
```

### Phase 2: Configuration (Week 2)

#### Driver Configuration

Create `driver/config.go`:

```go
package driver

import "github.com/hashicorp/nomad/plugins/shared/hclspec"

var (
    // configSpec is the HCL specification for the driver config
    configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
        "enabled": hclspec.NewDefault(
            hclspec.NewAttr("enabled", "bool", false),
            hclspec.NewLiteral("true"),
        ),
        "elide_binary": hclspec.NewDefault(
            hclspec.NewAttr("elide_binary", "string", false),
            hclspec.NewLiteral(`"/usr/local/bin/elide"`),
        ),
    })
    
    // taskConfigSpec is the HCL specification for task config
    taskConfigSpec = hclspec.NewObject(map[string]*hclspec.Spec{
        "script": hclspec.NewAttr("script", "string", true),
        "args": hclspec.NewAttr("args", "list(string)", false),
        "env": hclspec.NewAttr("env", "map(string)", false),
        "elide_opts": hclspec.NewBlock("elide_opts", false, hclspec.NewObject(map[string]*hclspec.Spec{
            "elide_binary": hclspec.NewAttr("elide_binary", "string", false),
            "languages": hclspec.NewAttr("languages", "list(string)", false),
            "memory_limit": hclspec.NewAttr("memory_limit", "number", false),
            "enable_ai": hclspec.NewAttr("enable_ai", "bool", false),
        })),
    })
)

// Config is the driver configuration set by the agent
type Config struct {
    Enabled      bool   `codec:"enabled"`
    ElideBinary  string `codec:"elide_binary"`
}

// TaskConfig is the per-task configuration
type TaskConfig struct {
    // Path to the Elide script or application
    Script string `codec:"script"`
    
    // Arguments to pass to the script
    Args []string `codec:"args"`
    
    // Environment variables
    Env map[string]string `codec:"env"`
    
    // Elide-specific options
    ElideOpts ElideOptions `codec:"elide_opts"`
}

// ElideOptions contains Elide-specific configuration
type ElideOptions struct {
    // Path to elide binary (overrides driver config)
    ElideBinary string `codec:"elide_binary"`
    
    // Guest languages to enable (e.g., ["python", "javascript"])
    Languages []string `codec:"languages"`
    
    // Memory limit for the VM in MB
    MemoryLimit int `codec:"memory_limit"`
    
    // Enable local AI inference
    EnableAI bool `codec:"enable_ai"`
}

// Validate checks if the task configuration is valid
func (tc *TaskConfig) Validate() error {
    if tc.Script == "" {
        return fmt.Errorf("script path is required")
    }
    
    if tc.ElideOpts.MemoryLimit < 0 {
        return fmt.Errorf("memory_limit must be positive")
    }
    
    return nil
}
```

#### Task Handle

Create `driver/handle.go`:

```go
package driver

import (
    "bufio"
    "io"
    "os/exec"
    "time"
    
    "github.com/hashicorp/go-hclog"
    "github.com/hashicorp/nomad/plugins/drivers"
)

// TaskHandle stores runtime information for a running task
type TaskHandle struct {
    taskID      string
    taskName    string
    pid         int
    command     *exec.Cmd
    startedAt   time.Time
    completedAt time.Time
    exitResult  *drivers.ExitResult
    logger      hclog.Logger
    
    // State management
    stateLock sync.RWMutex
    state     drivers.TaskState
}

// collectLogs reads from a pipe and logs the output
func (h *TaskHandle) collectLogs(stream string, reader io.Reader) {
    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
        line := scanner.Text()
        h.logger.Info("task output", "stream", stream, "line", line)
    }
}

// monitorExit waits for the process to exit and records the result
func (h *TaskHandle) monitorExit() {
    err := h.command.Wait()
    
    h.stateLock.Lock()
    defer h.stateLock.Unlock()
    
    h.completedAt = time.Now()
    h.state = drivers.TaskStateExited
    
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            h.exitResult = &drivers.ExitResult{
                ExitCode: exitErr.ExitCode(),
                Signal:   0,
                Err:      err,
            }
        } else {
            h.exitResult = &drivers.ExitResult{
                ExitCode: -1,
                Err:      err,
            }
        }
    } else {
        h.exitResult = &drivers.ExitResult{
            ExitCode: 0,
        }
    }
    
    h.logger.Info("task exited", "exit_code", h.exitResult.ExitCode)
}

// getState returns the current state of the task
func (h *TaskHandle) getState() drivers.TaskState {
    h.stateLock.RLock()
    defer h.stateLock.RUnlock()
    return h.state
}

// emitStats periodically collects and emits resource usage statistics
func (h *TaskHandle) emitStats(ctx context.Context, ch chan *drivers.TaskResourceUsage, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    defer close(ch)
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            stats, err := h.collectStats()
            if err != nil {
                h.logger.Warn("failed to collect stats", "error", err)
                continue
            }
            
            select {
            case ch <- stats:
            case <-ctx.Done():
                return
            }
        }
    }
}

// collectStats gathers current resource usage
func (h *TaskHandle) collectStats() (*drivers.TaskResourceUsage, error) {
    // TODO: Implement actual stats collection
    // For now, return dummy stats
    return &drivers.TaskResourceUsage{
        ResourceUsage: &drivers.ResourceUsage{
            CpuStats: &drivers.CpuStats{
                SystemMode: 0,
                UserMode:   0,
                TotalTicks: 0,
            },
            MemoryStats: &drivers.MemoryStats{
                RSS:      0,
                Cache:    0,
                Swap:     0,
                MaxUsage: 0,
            },
        },
        Timestamp: time.Now().UnixNano(),
    }, nil
}
```

### Phase 3: Protocol Buffers Integration (Week 3)

#### Set Up Buf

Create `proto/buf.yaml`:
```yaml
version: v1

name: buf.build/elide-dev/task-driver

deps:
  # Depend on published Elide protos
  - buf.build/elide/elide

build:
  excludes:
    - gen
    - vendor

lint:
  use:
    - DEFAULT
```

Create `proto/buf.gen.yaml`:
```yaml
version: v1

managed:
  enabled: true
  go_package_prefix:
    default: github.com/elide-dev/elide-task-driver/proto/gen/go

plugins:
  # Generate Go code
  - plugin: go
    out: gen/go
    opt:
      - paths=source_relative
      
  # Generate gRPC service stubs
  - plugin: go-grpc
    out: gen/go
    opt:
      - paths=source_relative
```

#### Generate Proto Code

Create `scripts/gen-proto.sh`:
```bash
#!/bin/bash
set -e

cd proto

# Update dependencies
buf dep update

# Generate Go code from Elide protos
buf generate --path ../elide/packages/proto

echo "Proto code generated successfully"
```

#### Use Generated APIs

Create `driver/elide.go`:
```go
package driver

import (
    "context"
    "fmt"
    "net"
    
    callapi "github.com/elide-dev/elide-task-driver/proto/gen/go/elide/call/v1alpha1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// ElideClient wraps gRPC communication with Elide runtime
type ElideClient struct {
    conn       *grpc.ClientConn
    callClient callapi.InvocationApiClient
}

// NewElideClient creates a client connected to Elide via Unix socket
func NewElideClient(socketPath string) (*ElideClient, error) {
    // Create Unix socket dialer
    dialer := func(ctx context.Context, addr string) (net.Conn, error) {
        return net.Dial("unix", addr)
    }
    
    // Connect via gRPC
    conn, err := grpc.Dial(
        socketPath,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithContextDialer(dialer),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to connect to elide: %v", err)
    }
    
    return &ElideClient{
        conn:       conn,
        callClient: callapi.NewInvocationApiClient(conn),
    }, nil
}

// Fetch invokes an HTTP-style handler in the Elide application
func (c *ElideClient) Fetch(ctx context.Context, req *callapi.FetchRequest) (*callapi.FetchResponse, error) {
    return c.callClient.Fetch(ctx, req)
}

// Close closes the gRPC connection
func (c *ElideClient) Close() error {
    return c.conn.Close()
}
```

---

## Communication Strategy

The Elide task driver uses **Unix sockets with gRPC** for all communication between the driver and Elide runtime instances. This provides type-safe, bidirectional communication with native Protocol Buffer serialization.

### Why Unix Sockets?

**Advantages**:
- **Type-safe gRPC communication** - No manual JSON parsing or serialization
- **Bidirectional streaming** - Driver can both send commands and receive events
- **High performance** - No TCP overhead, direct kernel communication
- **Native protobuf** - Automatic serialization/deserialization via generated code
- **Secure** - No network exposure, filesystem permissions control access
- **Efficient** - Lower latency than network sockets or HTTP

### Architecture Overview

```
┌─────────────────┐                    ┌──────────────────┐
│  Nomad Client   │                    │  Elide Runtime   │
│                 │                    │                  │
│  ┌───────────┐  │                    │  ┌────────────┐  │
│  │  Driver   │  │  Unix Socket       │  │ gRPC Server│  │
│  │  (Go)     │──┼───────────────────►│  │ (Kotlin)   │  │
│  └───────────┘  │  /tmp/elide-*.sock │  └────────────┘  │
│                 │                    │                  │
│  gRPC Client    │◄──────────────────►│  InvocationApi   │
│  (generated)    │     Protobuf       │  (generated)     │
└─────────────────┘                    └──────────────────┘
```

### Implementation

#### Step 1: Configure Elide to Listen on Unix Socket

In `driver/driver.go`, when starting a task:

```go
func (d *ElideDriver) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
    // 1. Generate unique socket path for this task
    socketPath := filepath.Join(
        cfg.AllocDir,
        fmt.Sprintf("elide-%s.sock", cfg.ID),
    )
    
    // 2. Build command with socket flag
    cmd := exec.Command(
        elideBinary,
        "run",
        "--socket", socketPath,  // Tell Elide to listen here
        taskConfig.Script,
    )
    
    // 3. Start Elide process
    if err := cmd.Start(); err != nil {
        return nil, nil, fmt.Errorf("failed to start elide: %v", err)
    }
    
    // 4. Wait for socket to be created
    if err := waitForSocket(socketPath, 5*time.Second); err != nil {
        return nil, nil, fmt.Errorf("elide socket not ready: %v", err)
    }
    
    // 5. Connect gRPC client to socket
    client, err := NewElideClient(socketPath)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to connect to elide: %v", err)
    }
    
    // 6. Store client in task handle for later use
    handle := &TaskHandle{
        taskID:      cfg.ID,
        pid:         cmd.Process.Pid,
        command:     cmd,
        socketPath:  socketPath,
        grpcClient:  client,
        startedAt:   time.Now(),
    }
    
    return &drivers.TaskHandle{...}, nil, nil
}

// waitForSocket polls until socket file exists
func waitForSocket(path string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if _, err := os.Stat(path); err == nil {
            return nil // Socket exists
        }
        time.Sleep(100 * time.Millisecond)
    }
    return fmt.Errorf("timeout waiting for socket")
}
```

#### Step 2: Create gRPC Client (Already Implemented in Phase 3)

The `ElideClient` from `driver/elide.go` handles the connection:

```go
// NewElideClient creates a client connected to Elide via Unix socket
func NewElideClient(socketPath string) (*ElideClient, error) {
    // Unix socket dialer
    dialer := func(ctx context.Context, addr string) (net.Conn, error) {
        return net.Dial("unix", addr)
    }
    
    // Connect via gRPC
    conn, err := grpc.Dial(
        socketPath,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithContextDialer(dialer),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %v", err)
    }
    
    return &ElideClient{
        conn:       conn,
        callClient: callapi.NewInvocationApiClient(conn),
    }, nil
}
```

#### Step 3: Make gRPC Calls

Once connected, you can invoke Elide applications:

```go
// Example: Invoke HTTP-style handler
func (h *TaskHandle) invokeHandler(ctx context.Context, path string, body []byte) error {
    req := &callapi.FetchRequest{
        Request: &httpapi.HttpRequest{
            Method: httpapi.HttpMethod_GET,
            Path:   path,
            Body:   body,
        },
    }
    
    resp, err := h.grpcClient.Fetch(ctx, req)
    if err != nil {
        return fmt.Errorf("fetch failed: %v", err)
    }
    
    h.logger.Info("response received", "status", resp.Response.StatusCode)
    return nil
}

// Example: Health check via gRPC
func (h *TaskHandle) healthCheck(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    
    return h.invokeHandler(ctx, "/health", nil)
}
```

### Socket Lifecycle Management

#### Creation
- **When**: During `StartTask()`
- **Where**: Task allocation directory (isolated per task)
- **Permissions**: 0600 (owner read/write only)

#### Monitoring
```go
// Check if socket is still valid
func (h *TaskHandle) isSocketAlive() bool {
    if _, err := os.Stat(h.socketPath); err != nil {
        return false
    }
    return true
}
```

#### Cleanup
```go
// In StopTask or DestroyTask
func (d *ElideDriver) DestroyTask(taskID string, force bool) error {
    handle, ok := d.tasks[taskID]
    if !ok {
        return nil
    }
    
    // 1. Close gRPC connection
    if handle.grpcClient != nil {
        handle.grpcClient.Close()
    }
    
    // 2. Stop Elide process
    if handle.command.Process != nil {
        handle.command.Process.Kill()
    }
    
    // 3. Remove socket file
    os.Remove(handle.socketPath)
    
    delete(d.tasks, taskID)
    return nil
}
```

### Error Handling

```go
// Handle socket connection errors
func (d *ElideDriver) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
    // ... start process ...
    
    // Retry connection with backoff
    var client *ElideClient
    var lastErr error
    
    for i := 0; i < 10; i++ {
        client, lastErr = NewElideClient(socketPath)
        if lastErr == nil {
            break
        }
        
        d.logger.Warn("failed to connect, retrying", "attempt", i+1, "error", lastErr)
        time.Sleep(time.Duration(i*100) * time.Millisecond)
    }
    
    if client == nil {
        cmd.Process.Kill()
        return nil, nil, fmt.Errorf("failed to connect after retries: %v", lastErr)
    }
    
    // ... continue ...
}
```

### Security Considerations

1. **Socket Permissions**: Sockets are created in the task's allocation directory with restrictive permissions
2. **No Network Exposure**: Unix sockets don't bind to network interfaces
3. **Process Isolation**: Each task gets its own socket, preventing cross-task communication
4. **Cleanup**: Sockets are automatically removed when tasks stop

### Performance Characteristics

| Metric | Unix Socket | TCP Loopback | HTTP |
|--------|-------------|--------------|------|
| Latency | ~10-20μs | ~50-100μs | ~200-500μs |
| Throughput | ~10GB/s | ~5GB/s | ~2GB/s |
| CPU Overhead | Low | Medium | High |
| Memory | Minimal | Low | Medium |

### Troubleshooting

#### Socket Not Created
```go
// Check if Elide process started
if !cmd.Process.Signal(syscall.Signal(0)) {
    return fmt.Errorf("elide process died before socket creation")
}

// Check socket path permissions
dir := filepath.Dir(socketPath)
if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
    return fmt.Errorf("socket directory not accessible: %v", err)
}
```

#### Connection Refused
```go
// Verify socket file exists and is a socket
if stat, err := os.Stat(socketPath); err != nil {
    return fmt.Errorf("socket file missing: %v", err)
} else if stat.Mode()&os.ModeSocket == 0 {
    return fmt.Errorf("not a socket: %s", socketPath)
}
```

#### gRPC Errors
```go
// Add interceptors for better error messages
opts := []grpc.DialOption{
    grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, 
        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        
        err := invoker(ctx, method, req, reply, cc, opts...)
        if err != nil {
            d.logger.Error("grpc call failed", "method", method, "error", err)
        }
        return err
    }),
}
```

---

## Testing

### End-to-End Testing with Stubbed Server

The driver has been successfully tested end-to-end with a stubbed gRPC server. This allows full testing without waiting for the real Elide daemon.

#### Test Setup (4 Terminals)

**Terminal 1: Stubbed Server**
```bash
cd /path/to/elide-task-driver
make server
```
Server listens on `/tmp/elide-daemon.sock`

**Terminal 2: Nomad Agent**
```bash
cd /path/to/elide-task-driver
nomad agent -dev -config=nomad-agent.hcl
```
Verify driver loaded:
```
[INFO]  agent: detected plugin: name=elide type=driver plugin_version=v0.1.0
[DEBUG] client.driver_mgr: detected drivers: drivers="map[healthy:[raw_exec elide java]...]"
```

**Terminal 3: Verify Driver**
```bash
nomad node status -self | grep -i driver
```
Should show: `Driver Status = elide,java,raw_exec`

**Terminal 4: Submit Test Job**
```bash
cd /path/to/elide-task-driver
nomad job run examples/hello-python.nomad
```

#### Expected Results

✅ **Job scheduled successfully**
- Allocation created
- Task started
- Exit code: 0
- Status: complete

✅ **Check job status:**
```bash
nomad job status hello-python
nomad alloc status <allocation-id>
nomad alloc logs <allocation-id>
```

#### What Gets Tested

1. **Plugin Loading** - Driver loads in Nomad
2. **Session Creation** - Driver creates session with daemon
3. **Task Execution** - Driver executes snippets via gRPC
4. **Status Polling** - Driver polls execution status
5. **Task Completion** - Driver reports completion correctly

### Configuration

The driver uses `nomad-agent.hcl` for configuration:

```hcl
plugin_dir = "/path/to/elide-task-driver/build/plugins"

plugin "elide" {
  config {
    daemon_socket = "/tmp/elide-daemon.sock"
    
    session_config {
      context_pool_size  = 10
      enabled_languages  = ["python", "javascript", "typescript"]
      enabled_intrinsics = ["io", "env"]
      memory_limit_mb    = 512
      enable_ai          = false
    }
  }
}
```

### Troubleshooting

**Driver not appearing:**
1. Check plugin symlink exists: `ls -lh build/plugins/elide`
2. Verify config file: `cat nomad-agent.hcl`
3. Check Nomad logs for plugin loading errors

**Connection errors:**
1. Verify stubbed server is running: `ls -la /tmp/elide-daemon.sock`
2. Check server logs in Terminal 1

**Session creation fails:**
1. Ensure stubbed server is running before starting Nomad
2. Check daemon_socket path in config matches server

---

## Example Job Specs

### Example 1: Hello Python

Create `examples/hello-python.nomad`:
```hcl
job "hello-python" {
  datacenters = ["dc1"]
  type = "batch"
  
  group "python" {
    count = 1
    
    task "hello" {
      driver = "elide"
      
      config {
        script = "local/hello.py"
        
        elide_opts {
          languages = ["python"]
        }
      }
      
      # Inline the script
      template {
        data = <<EOF
#!/usr/bin/env python3
print("Hello from Elide on Nomad!")
print("This is a polyglot runtime")
EOF
        destination = "local/hello.py"
        perms = "755"
      }
      
      resources {
        cpu    = 100
        memory = 128
      }
    }
  }
}
```

### Example 2: HTTP Server

Create `examples/http-server.nomad`:
```hcl
job "elide-web" {
  datacenters = ["dc1"]
  type = "service"
  
  group "web" {
    count = 3
    
    network {
      port "http" {
        to = 8080
      }
    }
    
    task "server" {
      driver = "elide"
      
      config {
        script = "local/server.py"
        args = ["--port", "${NOMAD_PORT_http}"]
        
        env = {
          "LOG_LEVEL" = "info"
        }
        
        elide_opts {
          languages = ["python", "javascript"]
          memory_limit = 512
        }
      }
      
      artifact {
        source = "https://example.com/app.tar.gz"
        destination = "local/"
      }
      
      service {
        name = "elide-web"
        port = "http"
        
        check {
          type     = "http"
          path     = "/health"
          interval = "10s"
          timeout  = "2s"
        }
      }
      
      resources {
        cpu    = 500
        memory = 512
      }
    }
  }
}
```

### Example 3: AI Inference

Create `examples/ai-inference.nomad`:
```hcl
job "ai-worker" {
  datacenters = ["dc1"]
  type = "service"
  
  group "inference" {
    count = 2
    
    task "ai" {
      driver = "elide"
      
      config {
        script = "local/inference.py"
        
        elide_opts {
          languages = ["python"]
          memory_limit = 2048
          enable_ai = true
        }
      }
      
      template {
        data = <<EOF
# AI inference script using Elide's local AI
import polyglot

# Load Elide's AI module via JavaScript
helper = polyglot.eval(language='js', string='''
  import llm from "elide:llm";
  export function infer(prompt) {
    const model = llm.huggingface({
      repo: "TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF",
      name: "tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
    });
    const params = llm.params();
    return llm.inferSync(params, model, prompt);
  }
''')

# Use it
result = helper.infer("What is Elide?")
print(result)
EOF
        destination = "local/inference.py"
      }
      
      resources {
        cpu    = 1000
        memory = 2048
      }
    }
  }
}
```

### Example 4: Polyglot Application

Create `examples/polyglot.nomad`:
```hcl
job "polyglot-demo" {
  datacenters = ["dc1"]
  
  group "app" {
    task "main" {
      driver = "elide"
      
      config {
        script = "local/main.py"
        
        elide_opts {
          languages = ["python", "javascript", "kotlin"]
          memory_limit = 1024
        }
      }
      
      # Python calls JavaScript calls Kotlin
      template {
        data = <<EOF
import polyglot

# Load JavaScript helper
js_helper = polyglot.eval(language='js', string='''
  export function processData(data) {
    return data.map(x => x * 2);
  }
''')

# Use JavaScript from Python
data = [1, 2, 3, 4, 5]
result = js_helper.processData(data)
print(f"Processed: {result}")
EOF
        destination = "local/main.py"
      }
    }
  }
}
```

---

## API Integration

### Using Buf.build to Share APIs

#### Publish Your Driver's Proto Definitions

```bash
# In your proto directory
buf push
```

This publishes to `buf.build/elide-dev/task-driver`

#### Consume Elide's Published Protos

```yaml
# proto/buf.yaml
deps:
  - buf.build/elide/elide  # Official Elide protos
```

```bash
# Generate Go code from Elide protos
buf generate --path elide/
```

#### Document APIs for Your Team

Create `docs/API.md`:
```markdown
# Elide Task Driver APIs

## gRPC Services Used

### InvocationApi (from elide.call.v1alpha1)

Used to invoke Elide applications remotely.

**Methods**:
- `Fetch(FetchRequest) → FetchResponse`: HTTP-style invocation
- `Scheduled(ScheduledInvocationRequest) → ScheduledInvocationResponse`: Cron-style
- `Queue(QueueInvocationRequest) → QueueInvocationResponse`: Message-based

**Example**:
```go
client := NewElideClient(socketPath)
resp, err := client.Fetch(ctx, &callapi.FetchRequest{
    // ... request
})
```

## Proto References

- **Elide Protos**: https://buf.build/elide/elide
- **Driver Protos**: https://buf.build/elide-dev/task-driver
```

---

## Success Criteria

### MVP Checklist

#### Week 1-2: Foundation
- [ ] Project structure created
- [ ] Go module initialized
- [ ] Dependencies installed
- [ ] Skeleton driver compiles
- [ ] Loads in Nomad without errors
- [ ] Basic logging works

#### Week 3-4: Core Functionality
- [ ] Can start simple Python script
- [ ] Can start simple JavaScript script
- [ ] Captures stdout/stderr
- [ ] Reports task started event
- [ ] Reports task completed event
- [ ] Can stop running task with SIGTERM
- [ ] Task status reporting works
- [ ] Basic error handling

#### Week 5-6: Integration
- [ ] Proto code generation works
- [ ] gRPC client connects to Elide (if using socket)
- [ ] Environment variables passed correctly
- [ ] Artifact downloads work
- [ ] Resource limits applied
- [ ] Unit tests passing
- [ ] Integration tests passing

### V1.0 Checklist

- [ ] Unix socket communication
- [ ] gRPC service calls working
- [ ] Health checks implemented
- [ ] Graceful shutdown (SIGTERM handling)
- [ ] Task recovery after driver restart
- [ ] Resource monitoring (CPU, memory)
- [ ] Multi-language configuration
- [ ] Full test coverage (>80%)
- [ ] E2E tests with Nomad
- [ ] Documentation complete
- [ ] Example jobs tested

### V2.0 Goals

- [ ] AI workload optimizations
- [ ] Hot reload support
- [ ] Metrics export (Prometheus)
- [ ] Nomad service mesh integration
- [ ] Volume mounting
- [ ] Advanced scheduling hints
- [ ] Production deployments

---

## Project Documentation

This project includes additional documentation:

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Detailed architecture design and one-daemon approach
- **[FEATURE_REQUEST.md](FEATURE_REQUEST.md)** - Feature request to Elide team for daemon mode
- **[RESPONSIBILITIES.md](RESPONSIBILITIES.md)** - Responsibilities and dependencies breakdown
- **[VERIFICATION.md](VERIFICATION.md)** - Verification results and assumptions
- **[FINDINGS.md](FINDINGS.md)** - Codebase analysis findings
- **[UNDERSTANDING.md](UNDERSTANDING.md)** - Architecture understanding confirmation

---

## Resources

### Official Documentation

1. **Nomad Plugin Development**
   - Task Driver Interface: https://developer.hashicorp.com/nomad/docs/deploy/task-driver
   - Plugin SDK: https://github.com/hashicorp/nomad/tree/main/plugins
   - go-plugin: https://github.com/hashicorp/go-plugin

2. **Elide Documentation**
   - Main Docs: https://docs.elide.dev
   - GitHub: https://github.com/elide-dev/elide
   - Proto Definitions: https://buf.build/elide/elide

3. **Protocol Buffers**
   - Buf CLI: https://buf.build/docs
   - gRPC Go: https://grpc.io/docs/languages/go/
   - Protobuf Go: https://protobuf.dev/getting-started/gotutorial/

### Reference Implementations

1. **Firecracker Task Driver** (Your Example)
   - Repo: https://github.com/cneira/firecracker-task-driver
   - Good for: Plugin structure, lifecycle management

2. **Docker Driver** (Official)
   - Code: https://github.com/hashicorp/nomad/tree/main/drivers/docker
   - Good for: Complete feature reference

3. **Exec Driver** (Official, Simple)
   - Code: https://github.com/hashicorp/nomad/tree/main/drivers/exec
   - Good for: Minimal implementation pattern

4. **Raw Exec Driver** (Official, Simplest)
   - Code: https://github.com/hashicorp/nomad/tree/main/drivers/rawexec
   - Good for: Understanding basics

### Community Resources

- **Nomad Community Forum**: https://discuss.hashicorp.com/c/nomad
- **Elide Discord**: (Check elide.dev for invite)
- **Go Learning**: https://go.dev/tour/

### Tools

- **Nomad**: `brew install nomad`
- **Buf**: `brew install bufbuild/buf/buf`
- **Go**: `brew install go`
- **Protoc**: `brew install protobuf`

---

## Contributing

### Development Workflow

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/my-feature
   ```
3. **Make changes and test**
   ```bash
   make test
   ```
4. **Commit with clear messages**
   ```bash
   git commit -m "Add support for X"
   ```
5. **Push and create PR**
   ```bash
   git push origin feature/my-feature
   ```

### Code Style

- Follow Go conventions (use `gofmt`)
- Add comments for exported functions
- Write tests for new features
- Update documentation

### Testing Requirements

- Unit tests must pass
- Integration tests should pass (may skip in CI)
- Add E2E test for major features

---

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

---

## Acknowledgments

- **HashiCorp**: For Nomad and the plugin framework
- **Elide Team**: For the incredible polyglot runtime
- **cneira**: For the firecracker-task-driver reference implementation

---

## Support

- **Elide Questions**: Check Elide docs or Discord
- **Nomad Questions**: HashiCorp Discuss forum
- **Driver Issues**: Open an issue on this repo
- **Internal (Elide Team)**: Slack #task-driver channel

---

**Built by the Elide team**

*Junior Developer Guide - Updated November 4, 2025*

