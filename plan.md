# Implementation Plan: Elide Daemon Mode & Nomad Task Driver

## Overview

This plan outlines the implementation strategy for building an Elide daemon mode (library-based) and the corresponding Nomad task driver plugin. We're going straight to production-ready implementation.

---

## Architecture Decision

### End Goal: Library-Based Daemon

**Why**: The end goal is a **programmatic daemon library** (similar to `ElideEmbedded`), not a CLI command. This provides:
- Embeddable service that can be started programmatically
- Better long-term maintenance
- Flexible integration (CLI wrapper optional)
- Production-ready architecture

**Structure**:
```
elide/packages/daemon/
  ├── ElideDaemon.kt          # Main daemon class (like ElideEmbedded)
  ├── DaemonServer.kt         # gRPC server implementation
  ├── SnippetExecutor.kt     # Execution engine wrapper
  └── ...
```

**Optional CLI Wrapper** (for testing only):
```
elide/packages/cli/
  └── cmd/dev/
      └── DaemonCommand.kt   # Thin CLI wrapper (testing only)
```

---

## How This Enables Sam's Sub-500 Line Go Script

### The Requirement

**Sam's Goal**: Build a **sub-500 line Go script** for the Nomad task driver that enables resource-efficient execution of multiple Elide code snippets in a single shared daemon instance (one daemon per Nomad client, not one per task).

### The Problem: Without Daemon Mode

**Without daemon mode**, the Go driver would need to manage Elide processes directly, resulting in **800-1200+ lines of complex code**:

**Example from `skeleton/td-demo/hello/driver.go`** (lines 343-421):
```go
// StartTask - Complex process management (200+ lines just for this)
func (d *HelloDriverPlugin) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
    // Need to:
    // 1. Start Elide process (one per task)
    // 2. Manage stdin/stdout pipes
    // 3. Parse output to track tasks
    // 4. Handle process lifecycle
    // 5. Manage multiple concurrent processes
    // 6. Handle errors, timeouts, crashes
    
    executorConfig := &executor.ExecutorConfig{
        LogFile:  filepath.Join(cfg.TaskDir().Dir, "executor.out"),
        LogLevel: "debug",
    }
    
    exec, pluginClient, err := executor.CreateExecutor(d.logger, d.nomadConfig, executorConfig)
    // ... complex process management ...
}
```

**Why So Complex Without Daemon**:
- Process management (start/stop/monitor Elide processes)
- REPL communication via stdin/stdout (complex parsing)
- Output tracking (match outputs to tasks)
- Error handling (process crashes, timeouts)
- State management (multiple processes)
- Recovery logic

**Estimated Lines**: **800-1200+ lines** (doesn't meet Sam's requirement)

### The Solution: With Daemon Mode

**With daemon mode**, the Go driver becomes a **simple gRPC client wrapper** (~300-400 lines):

**Simple Implementation** (based on `skeleton/td-demo` structure):
```go
// StartTask - Simple gRPC call (~50 lines)
func (d *ElideDriver) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
    // 1. Read script file
    code := readFile(cfg.Script)
    
    // 2. Connect to Elide daemon (one per Nomad client)
    if d.daemonClient == nil {
        conn, err := grpc.Dial(d.config.DaemonSocket, grpc.WithInsecure())
        if err != nil {
            return nil, nil, fmt.Errorf("failed to connect to daemon: %v", err)
        }
        d.daemonClient = executionapi.NewExecutionApiClient(conn)
    }
    
    // 3. Call ExecuteSnippet gRPC
    resp, err := d.daemonClient.ExecuteSnippet(ctx, &executionapi.ExecuteSnippetRequest{
        Code:        code,
        Language:    cfg.Language,
        Env:         cfg.Env,
        ExecutionId: cfg.ID,
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to execute snippet: %v", err)
    }
    
    // 4. Create handle
    handle := drivers.NewTaskHandle(taskHandleVersion)
    handle.Config = cfg
    
    h := &taskHandle{
        executionId: resp.ExecutionId,
        status:      resp.Status,
        taskConfig:  cfg,
        startedAt:   time.Now(),
        logger:      d.logger,
    }
    
    d.tasks.Set(cfg.ID, h)
    return handle, nil, nil
}
```

**Why So Simple With Daemon**:
- Simple gRPC client calls (no process management)
- No complex parsing (structured responses)
- Daemon handles complexity (process lifecycle, execution engine)
- Clear API surface (type-safe via Protocol Buffers)

**Estimated Lines**: **300-400 lines** ✅ (meets Sam's requirement)

### Code Structure Comparison

**Reference**: `skeleton/td-demo/hello/driver.go` provides the base structure for Nomad drivers.

**Without Daemon** (Complex):
```
driver.go (~800-1200 lines)
├── Process management (~200 lines)
│   ├── StartElideProcess()
│   ├── StopElideProcess()
│   └── MonitorElideProcess()
├── REPL communication (~300 lines)
│   ├── SendSnippetViaStdin()
│   ├── ParseOutput()
│   └── TrackExecutionState()
├── Output parsing (~200 lines)
│   ├── ExtractTaskId()
│   ├── ParseStdout()
│   └── ParseStderr()
├── Error handling (~100 lines)
├── State management (~100 lines)
└── Recovery logic (~100 lines)
```

**With Daemon** (Simple):
```
driver.go (~300-400 lines)
├── gRPC client (~50 lines)
│   ├── ConnectToDaemon()
│   └── DaemonClient struct
├── Task lifecycle (~150 lines)
│   ├── StartTask() - gRPC call
│   ├── StopTask() - gRPC call
│   ├── InspectTask() - gRPC call
│   └── WaitTask() - gRPC polling
├── Task handle (~50 lines)
│   └── TaskStatus() - simple state
└── Nomad interface (~50 lines)
    ├── TaskConfigSchema()
    ├── RecoverTask()
    └── SignalTask()
```

### Key Files from `skeleton/td-demo` Reference

**Base Structure** (from `skeleton/td-demo/`):
- `main.go` - Plugin entry point (stays similar)
- `hello/driver.go` - Driver implementation (simplified with daemon)
- `hello/handle.go` - Task handle management (simplified)
- `hello/state.go` - State structures (simplified)
- `go.mod` - Dependencies (adds gRPC)

**What Changes**:
- **Simplified**: `StartTask()` becomes gRPC call instead of process management (see `skeleton/td-demo/hello/driver.go:343`)
- **Simplified**: `StopTask()` becomes gRPC call instead of process termination
- **Simplified**: `InspectTask()` becomes gRPC call instead of output parsing
- **Simplified**: Task handle stores `executionId` instead of `pid` and process state (see `skeleton/td-demo/hello/handle.go:21-36`)
- **Added**: gRPC client connection management
- **Removed**: Process management, REPL communication, output parsing

### The Connection

**Elide Daemon** (Kotlin):
- Handles all complexity (process management, execution engine, GraalVM)
- Provides simple gRPC API
- Manages shared GraalVM context
- Executes multiple snippets concurrently

**Nomad Driver** (Go):
- Simple gRPC client wrapper
- Calls daemon API
- Manages Nomad task lifecycle
- **Sub-500 lines** ✅

**Result**: Clean separation of concerns - daemon handles complexity, driver is simple bridge to Nomad.

---

## Prerequisites

### Local Development (No VPN Needed Yet)

**Required Tools**:
- ✅ Java 23+ (GraalVM)
- ✅ Gradle
- ✅ Kotlin compiler
- ✅ Go 1.21+ (for Nomad driver)
- ✅ Protocol Buffers compiler (for gRPC)
- ✅ Elide repo (already cloned locally)
- ✅ Nomad driver repo (already exists)

**VPN Requirements**:
- ❌ **Not needed** for local development, building, testing
- ✅ **Needed** when:
  - Pushing to company Git repositories (if private)
  - Accessing internal CI/CD systems
  - Accessing internal documentation/wiki
  - Accessing internal package registries (Maven, npm)

### Verify Local Setup

```bash
cd elide
make setup              # Install dependencies
./gradlew build         # Test build
```

---

## Implementation Plan

### Phase 1: Elide Daemon Library

#### 1.1 Create Daemon Package Structure

**Location**: `elide/packages/daemon/`

**Files to Create**:
- `build.gradle.kts` - Package build configuration
- `src/main/kotlin/elide/daemon/ElideDaemon.kt` - Main daemon class
- `src/main/kotlin/elide/daemon/DaemonServer.kt` - gRPC server
- `src/main/kotlin/elide/daemon/SnippetExecutor.kt` - Execution engine
- `src/main/kotlin/elide/daemon/DaemonConfig.kt` - Configuration
- `src/main/kotlin/elide/daemon/ExecutionState.kt` - State management

#### 1.2 Define gRPC Proto Definitions

**Location**: `elide/packages/proto/elide/daemon/v1alpha1/execution_api.proto`

**Required RPCs**:
```protobuf
service ExecutionApi {
  // Execute an arbitrary code snippet
  rpc ExecuteSnippet(ExecuteSnippetRequest) returns (ExecuteSnippetResponse);
  
  // Get execution status
  rpc GetExecutionStatus(GetExecutionStatusRequest) returns (GetExecutionStatusResponse);
  
  // Cancel execution
  rpc CancelExecution(CancelExecutionRequest) returns (CancelExecutionResponse);
  
  // Health check
  rpc Health(HealthRequest) returns (HealthResponse);
}
```

**Messages**:
- `ExecuteSnippetRequest` - code, language, env vars, execution_id
- `ExecuteSnippetResponse` - execution_id, status
- `GetExecutionStatusRequest` - execution_id
- `GetExecutionStatusResponse` - status, stdout, stderr, exit_code
- `CancelExecutionRequest` - execution_id
- `CancelExecutionResponse` - success

#### 1.3 Implement ElideDaemon Class

**Similar to `ElideEmbedded`**:
- Lifecycle management (initialize, start, stop)
- Shared GraalVM engine instance
- Thread-local context management
- Execution state tracking
- Resource cleanup

**Key Methods**:
```kotlin
class ElideDaemon {
  fun initialize(config: DaemonConfig): Boolean
  fun start(): Boolean
  fun stop(): Boolean
  suspend fun executeSnippet(request: ExecuteSnippetRequest): ExecuteSnippetResponse
  suspend fun getExecutionStatus(executionId: String): ExecutionStatus
  suspend fun cancelExecution(executionId: String): Boolean
}
```

#### 1.4 Implement SnippetExecutor

**Responsibilities**:
- Execute arbitrary code snippets (Python/JS/TS)
- Capture stdout/stderr
- Track execution state
- Handle timeouts
- Isolate execution contexts

**Implementation Points**:
- Use Elide's existing execution infrastructure
- Reuse `ToolShellCommand` execution logic
- Manage execution lifecycle
- Return structured results

#### 1.5 Implement gRPC Server

**Responsibilities**:
- Start/stop gRPC server
- Handle incoming requests
- Route to execution engine
- Manage connections
- Error handling

**Transport Options**:
- Unix socket (recommended for local)
- TCP (for remote)
- Both (configurable)

#### 1.6 Add to Elide Build System

**Update**:
- `elide/settings.gradle.kts` - Add daemon package
- `elide/build.gradle.kts` - Include daemon in build
- `elide/packages/bom/build.gradle.kts` - Add to BOM if needed

---

### Phase 2: Nomad Task Driver

#### 2.1 Base Implementation (Using td-demo)

**Location**: `nomad-driver/elide-task-driver/`

**Files to Create/Modify**:
- `main.go` - Plugin entry point
- `driver/driver.go` - Main driver logic
- `driver/config.go` - Configuration schema
- `driver/daemon_client.go` - gRPC client for Elide daemon
- `driver/task_handle.go` - Task lifecycle management
- `go.mod` - Dependencies
- `build.gradle.kts` or `Makefile` - Build script

#### 2.2 Generate Go gRPC Stubs

**From Proto Definitions**:
```bash
# Generate Go code from proto
protoc --go_out=. --go-grpc_out=. \
  elide/packages/proto/elide/daemon/v1alpha1/execution_api.proto
```

**Dependencies**:
- `google.golang.org/grpc`
- `google.golang.org/protobuf`
- `github.com/hashicorp/nomad/plugins/drivers`

#### 2.3 Implement Driver Core Methods

**Required Nomad Driver Interface Methods**:

1. **TaskConfigSchema()** - Define HCL schema for task config
   ```hcl
   script = "path/to/script.py"  # or inline code
   language = "python"           # or "javascript", "typescript"
   env = {                       # environment variables
     KEY = "value"
   }
   daemon_socket = "/tmp/elide-daemon.sock"  # optional
   ```

2. **StartTask()** - Start a task
   - Connect to Elide daemon (ensure it's running)
   - Prepare code snippet (read file or use inline)
   - Call `ExecuteSnippet` gRPC
   - Create task handle
   - Return handle to Nomad

3. **StopTask()** - Stop a task
   - Call `CancelExecution` gRPC
   - Clean up task handle
   - Return success

4. **InspectTask()** - Get task status
   - Call `GetExecutionStatus` gRPC
   - Return status to Nomad

5. **WaitTask()** - Wait for task completion
   - Poll `GetExecutionStatus` until complete
   - Return exit result

6. **RecoverTask()** - Recover task after restart
   - Reconnect to Elide daemon
   - Reattach to execution
   - Return handle

#### 2.4 Daemon Lifecycle Management

**Responsibilities**:
- Ensure Elide daemon is running (one per Nomad client)
- Start daemon if not running
- Manage daemon process lifecycle
- Handle daemon crashes/restarts

**Implementation Options**:
- **Option A**: Start daemon as subprocess from driver
  - Driver spawns `elide daemon` process
  - Manages daemon lifecycle
  - Simple but requires Elide CLI

- **Option B**: Embed daemon library (if possible)
  - Use Elide daemon as Go library (via JNI/CGO)
  - More complex but better integration
  - Requires JVM in process

- **Option C**: Expect daemon to be pre-started
  - Systemd/service manages daemon
  - Driver just connects
  - Simplest for production

**Recommendation**: Start with Option A (MVP), move to Option C (production)

#### 2.5 Task Handle Management

**Store**:
- Execution ID from Elide daemon
- Task configuration
- Start time
- Process state
- gRPC connection state

---

### Phase 3: Integration & Testing

#### 3.1 Local Development Testing

**Test Daemon Library**:
```bash
cd elide
./gradlew :packages:daemon:test
```

**Test Nomad Driver**:
```bash
cd nomad-driver/elide-task-driver
make build
nomad agent -dev -plugin-dir=$(pwd)
```

**End-to-End Test**:
1. Start Elide daemon
2. Start Nomad dev agent with driver
3. Submit test job
4. Verify execution
5. Check logs

#### 3.2 Test Scenarios

**Basic Execution**:
- Python script execution
- JavaScript snippet execution
- TypeScript code execution
- Environment variable passing
- stdout/stderr capture

**Error Handling**:
- Script errors
- Timeout handling
- Daemon disconnection
- Task cancellation

**Concurrency**:
- Multiple tasks simultaneously
- Shared daemon instance
- Resource isolation

**Lifecycle**:
- Task start/stop
- Driver restart recovery
- Daemon restart recovery

---

### Phase 4: Documentation & Deployment

#### 4.1 Documentation

**Elide Daemon**:
- API documentation
- Usage examples
- Configuration guide

**Nomad Driver**:
- Installation guide
- Configuration reference
- Example job specs
- Troubleshooting guide

#### 4.2 Deployment

**Elide Daemon**:
- Package as library (JAR)
- Optional: Native image
- Optional: CLI wrapper

**Nomad Driver**:
- Build Go plugin binary
- Distribute via GitHub releases
- Installation instructions

---

## Development Workflow

### Step 1: Verify Local Setup

```bash
# Verify Elide builds
cd elide
make setup
./gradlew build

# Verify Go setup
cd ../elide-task-driver
go version
go mod download
```

### Step 2: Implement Daemon Library

1. Create `elide/packages/daemon/` package
2. Implement proto definitions
3. Implement `ElideDaemon` class
4. Implement gRPC server
5. Test locally

### Step 3: Implement Nomad Driver

1. Generate Go gRPC stubs
2. Implement driver core methods
3. Implement daemon client
4. Test locally with Nomad dev agent

### Step 4: Integration Testing

1. Start Elide daemon
2. Start Nomad with driver
3. Submit test jobs
4. Verify end-to-end

### Step 5: Get VPN (When Ready)

- Push code to company repos
- Access internal resources
- Collaborate with Elide team

---

## Timeline Estimate

### Phase 1: Daemon Library (2-3 weeks)
- Week 1: Package structure, proto definitions, basic daemon class
- Week 2: Execution engine, gRPC server
- Week 3: Testing, integration

### Phase 2: Nomad Driver (1-2 weeks)
- Week 1: Driver implementation, gRPC client
- Week 2: Testing, integration

### Phase 3: Integration & Testing (1 week)
- End-to-end testing
- Bug fixes
- Documentation

**Total**: ~4-6 weeks

---

## Success Criteria

### Daemon Library
- ✅ Can execute arbitrary code snippets (Python/JS/TS)
- ✅ Multiple concurrent executions in shared context
- ✅ gRPC API for programmatic access
- ✅ Proper lifecycle management
- ✅ Error handling and recovery

### Nomad Driver
- ✅ Sub-500 lines of Go code (simple gRPC client wrapper)
- ✅ Full Nomad driver interface implemented
- ✅ One daemon per Nomad client
- ✅ Multiple tasks per daemon
- ✅ Proper task lifecycle management
- ✅ Recovery after restarts

### Integration
- ✅ End-to-end test passes
- ✅ Multiple tasks run concurrently
- ✅ Resource efficiency (one GraalVM instance)
- ✅ Production-ready

---

## Key Design Decisions

### 1. Library vs CLI
- **Decision**: Library-based daemon (like `ElideEmbedded`)
- **Rationale**: Better for production, embeddable, flexible

### 2. gRPC vs HTTP
- **Decision**: gRPC (with optional HTTP)
- **Rationale**: Type-safe, efficient, structured responses

### 3. Daemon Lifecycle
- **Decision**: One daemon per Nomad client (managed by driver or system)
- **Rationale**: Resource efficiency, shared GraalVM context

### 4. Execution Isolation
- **Decision**: Isolated execution contexts within shared engine
- **Rationale**: Safety while maintaining efficiency

### 5. Driver Complexity
- **Decision**: Simple gRPC client wrapper (~300-400 lines)
- **Rationale**: Daemon handles complexity, driver just bridges to Nomad

---

## Next Steps

1. ✅ **Verify local build setup** (Elide + Go)
2. ⏭️ **Create daemon package structure**
3. ⏭️ **Define proto definitions**
4. ⏭️ **Implement `ElideDaemon` class**
5. ⏭️ **Implement gRPC server**
6. ⏭️ **Test daemon library**
7. ⏭️ **Implement Nomad driver**
8. ⏭️ **End-to-end testing**
9. ⏭️ **Get VPN when ready to push**

---

## Questions & Notes

### Open Questions
- Should daemon be started as subprocess or expected to be running?
- Unix socket vs TCP for gRPC transport?
- Should we support both?

### Notes
- Can reuse `ElideEmbedded` as reference for lifecycle management
- Can reuse `ToolShellCommand` execution logic
- Can reference `McpCommand` and `LspCommand` for long-running services
- Nomad driver should be simple - daemon handles complexity

---

## Resources

### Elide Codebase References
- `elide/packages/embedded/ElideEmbedded.kt` - Lifecycle management pattern
- `elide/packages/cli/src/main/kotlin/elide/tool/cli/cmd/repl/ToolShellCommand.kt` - Execution logic
- `elide/packages/cli/src/main/kotlin/elide/tool/cli/cmd/dev/McpCommand.kt` - Long-running service example
- `elide/packages/proto/elide/call/v1alpha1/call_api.proto` - Existing proto structure

### Nomad References
- `nomad-driver/skeleton/td-demo/` - Driver skeleton
- [Nomad Driver Plugin Documentation](https://developer.hashicorp.com/nomad/docs/plugins/drivers)

### gRPC References
- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers Guide](https://protobuf.dev/getting-started/gotutorial/)

---

**Last Updated**: 2025-01-XX
**Status**: Planning Phase

