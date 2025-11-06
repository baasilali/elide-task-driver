# API Questions for Elide Daemon

This document tracks features and capabilities that are currently undefined or not yet supported by the Elide daemon API. These questions need to be answered when integrating with the real daemon implementation.

---

## Status: Awaiting Real Daemon Implementation

The driver currently works end-to-end with a **stubbed server** that mocks these features. When the real Elide daemon is ready, we'll need clarification on which of these capabilities are supported.

---

## 1. Per-Execution Resource Metrics

**Current Status**: Not implemented (TaskStats returns empty channel)

**Question**: Does the daemon expose per-execution resource usage metrics?

**What We Need**:
- CPU usage (system time, user time, total ticks)
- Memory usage (RSS, cache, swap, max usage)
- Real-time or periodic metrics collection
- Metrics scoped to individual executions (not just daemon-wide)

**Driver Impact**:
- `TaskStats()` method currently returns empty stats channel
- Nomad scheduler can't enforce resource limits without metrics
- Can't detect runaway tasks or memory leaks

**API Suggestion**:
```protobuf
message GetExecutionMetricsRequest {
  string session_id = 1;
  string execution_id = 2;
}

message GetExecutionMetricsResponse {
  CpuMetrics cpu = 1;
  MemoryMetrics memory = 2;
  int64 timestamp = 3;
}
```

**Related Code**:
- `driver/driver.go:509-529` - TaskStats implementation
- TODO comment: "Once daemon API is available, collect stats from daemon"

---

## 2. Signal Forwarding to Executions

**Current Status**: Not implemented (SignalTask logs warning)

**Question**: Can the daemon forward Unix signals to running executions?

**What We Need**:
- Forward SIGTERM for graceful shutdown
- Forward SIGINT for interruption
- Forward SIGKILL for forced termination
- Signal handling per execution (not session-wide)

**Current Behavior**:
- `CancelExecution` provides binary stop (no signal control)
- Tasks can't distinguish between graceful shutdown vs forced kill
- Applications can't perform cleanup on SIGTERM

**Driver Impact**:
- `SignalTask()` method currently does nothing
- `StopTask()` uses `CancelExecution` (binary stop)
- No graceful shutdown for long-running tasks

**API Suggestion**:
```protobuf
message CancelExecutionRequest {
  string session_id = 1;
  string execution_id = 2;
  string signal = 3;  // "SIGTERM", "SIGKILL", etc.
  int32 timeout_seconds = 4;  // For graceful -> force escalation
}
```

**Related Code**:
- `driver/driver.go:537-550` - SignalTask implementation
- `driver/driver.go:464-480` - StopTask uses CancelExecution
- TODO comment: "Forward signal to execution if daemon supports it"

---

## 3. Per-Task Configuration Overrides

**Current Status**: Defined but not used (reserved for future use)

**Question**: Can individual executions override session-level configuration?

**What We Need**:
- Per-execution memory limits (override session default)
- Per-execution AI enablement (override session default)
- Per-execution execution timeout
- Per-execution enabled intrinsics

**Current Behavior**:
- All tasks use session-level configuration
- `TaskConfig.ElideOptions` are defined but ignored
- No way to give one task more memory than another

**Driver Impact**:
- Users can't set per-task resource limits
- All tasks in a session have identical capabilities
- Can't run high-memory task alongside low-memory tasks

**API Suggestion**:
```protobuf
message ExecuteSnippetRequest {
  string session_id = 1;
  string execution_id = 2;
  string code = 3;
  string language = 4;
  map<string, string> env = 5;
  repeated string args = 6;
  
  // Per-execution overrides (optional)
  ExecutionConfiguration config = 7;
}

message ExecutionConfiguration {
  optional uint64 memory_limit_mb = 1;
  optional bool enable_ai = 2;
  optional int32 timeout_seconds = 3;
  repeated string enabled_intrinsics = 4;
}
```

**Related Code**:
- `driver/config.go:74-86` - ElideOptions in taskConfigSpec (marked as not used)
- `driver/config.go:124-132` - ElideOptions struct with "RESERVED FOR FUTURE USE" comment
- `driver/driver.go:310-323` - StartTask doesn't use ElideOptions

---

## 4. Execution Streaming (stdout/stderr)

**Current Status**: Polled via GetExecutionStatus

**Question**: Does the daemon support streaming stdout/stderr as execution runs?

**What We Need**:
- Real-time log streaming (not just at completion)
- Separate stdout and stderr streams
- Line-buffered or block-buffered output
- Backpressure handling if consumer is slow

**Current Behavior**:
- Driver polls `GetExecutionStatus` every 1 second
- stdout/stderr only available when execution completes
- No real-time logs during execution
- High latency for long-running tasks

**Driver Impact**:
- Users can't see logs until task completes
- Can't debug running tasks
- Poor UX for long-running jobs
- Polling is inefficient (3,600 requests/hour per task)

**API Suggestion**:
```protobuf
service ExecutionApi {
  // ... existing methods ...
  
  // Stream execution output in real-time
  rpc StreamExecutionOutput(StreamExecutionOutputRequest) 
      returns (stream StreamExecutionOutputResponse);
}

message StreamExecutionOutputRequest {
  string session_id = 1;
  string execution_id = 2;
  bool include_stdout = 3;
  bool include_stderr = 4;
}

message StreamExecutionOutputResponse {
  string stream = 1;  // "stdout" or "stderr"
  bytes data = 2;
  int64 timestamp = 3;
}
```

**Alternative**: Server-sent events or WebSocket for streaming

**Related Code**:
- `driver/driver.go:420-459` - handleWait polls every 1 second
- TODO comment: "Consider exponential backoff or streaming"

---

## 5. Execution Status Notifications (Push vs Poll)

**Current Status**: Polling-based (WaitTask polls every 1 second)

**Question**: Can the daemon push status changes instead of requiring polling?

**What We Need**:
- Notification when execution completes
- Notification when execution fails
- Notification when execution is cancelled
- Event-driven model instead of polling

**Current Behavior**:
- Driver polls every 1 second
- Wastes resources for quick tasks
- Adds latency to completion detection
- Scale issues with many concurrent tasks

**Driver Impact**:
- Inefficient at scale (many tasks × polling frequency)
- 1-second latency before detecting completion
- Unnecessary load on daemon

**API Suggestion** (Option 1 - gRPC Streaming):
```protobuf
service ExecutionApi {
  // ... existing methods ...
  
  // Subscribe to execution status changes
  rpc WatchExecution(WatchExecutionRequest) 
      returns (stream ExecutionStatusEvent);
}

message WatchExecutionRequest {
  string session_id = 1;
  string execution_id = 2;
}

message ExecutionStatusEvent {
  string execution_id = 1;
  ExecutionStatus status = 2;
  bool complete = 3;
  int32 exit_code = 4;
  string error = 5;
  int64 timestamp = 6;
}
```

**API Suggestion** (Option 2 - Callback):
- Driver registers webhook URL with daemon
- Daemon calls webhook when execution status changes
- Less efficient for local communication

**Related Code**:
- `driver/driver.go:420-459` - handleWait polling loop
- `driver/driver.go:34` - fingerprintPeriod = 30s (less frequent but similar issue)

---

## 6. Intrinsics Configuration

**Current Status**: Hardcoded defaults, not validated by daemon

**Question**: What intrinsics are available and how are they configured?

**What We Need**:
- List of available intrinsics (io, env, fs, http, sqlite, ai, etc.)
- Which intrinsics are safe for multi-tenant environments
- Minimal set for maximum sandboxing
- Per-session or per-execution intrinsics control

**Current Behavior**:
- Driver defaults to `["io", "env"]`
- Based on CTO guidance: "start with smallest possible amount"
- No validation that daemon supports these
- No discovery of available intrinsics

**Driver Impact**:
- Can't validate intrinsics configuration
- Can't discover what's available
- Users might request unsupported intrinsics

**API Suggestion**:
```protobuf
service ExecutionApi {
  // ... existing methods ...
  
  // Get list of available intrinsics
  rpc GetAvailableIntrinsics(GetAvailableIntrinsicsRequest) 
      returns (GetAvailableIntrinsicsResponse);
}

message GetAvailableIntrinsicsResponse {
  repeated IntrinsicInfo intrinsics = 1;
}

message IntrinsicInfo {
  string name = 1;
  string description = 2;
  bool safe_for_multi_tenant = 3;
  repeated string required_permissions = 4;
}
```

**Related Code**:
- `driver/config.go:43-46` - enabled_intrinsics default to ["io", "env"]
- `driver/driver.go:164-167` - Uses defaults without validation
- `references/DARIO_FEEDBACK.md:125-135` - Guidance on minimal intrinsics

---

## 7. Execution Timeout Enforcement

**Current Status**: Not implemented

**Question**: Can the daemon enforce execution timeouts?

**What We Need**:
- Per-execution timeout (seconds or duration)
- Timeout behavior (cancel, signal, kill)
- Timeout status in execution result
- Grace period for cleanup

**Current Behavior**:
- No timeout enforcement
- Tasks can run indefinitely
- No way to limit execution time
- `ElideOptions.Timeout` is defined but unused

**Driver Impact**:
- Can't prevent runaway tasks
- Can't enforce SLAs
- Resource leaks from hung tasks

**API Suggestion**:
```protobuf
message ExecuteSnippetRequest {
  // ... existing fields ...
  int32 timeout_seconds = 7;
}

message GetExecutionStatusResponse {
  // ... existing fields ...
  bool timed_out = 9;
}
```

**Related Code**:
- `driver/config.go:85` - timeout defined in ElideOptions (not used)
- `driver/config.go:131` - Timeout field marked "not yet supported"

---

## 8. Session Lifecycle and Persistence

**Current Status**: Sessions created on driver init, may not persist

**Question**: How long do sessions live? What happens on daemon restart?

**What We Need**:
- Session persistence across daemon restarts
- Session cleanup on driver shutdown (implemented)
- Session expiry/timeout behavior
- Maximum sessions per daemon

**Current Behavior**:
- Driver creates session in `SetConfig`
- Driver deletes session in `Shutdown` (newly added)
- No handling of daemon restart
- No session recovery after connection loss

**Driver Impact**:
- Lost sessions if daemon restarts
- Tasks can't be recovered after daemon restart
- Need to recreate sessions after reconnection

**Questions**:
1. Are sessions persistent or in-memory?
2. Can sessions be recovered after daemon restart?
3. Is there a session TTL or timeout?
4. What happens to running executions if session is deleted?

**Related Code**:
- `driver/driver.go:145-190` - SetConfig creates session
- `driver/driver.go:560-585` - Shutdown deletes session (newly added)
- `driver/driver.go:353-407` - RecoverTask attempts to reconnect

---

## Summary Table

| Feature | Current Status | Blocking Issue | Priority | API Change Needed |
|---------|---------------|----------------|----------|-------------------|
| **Resource Metrics** | Not implemented | Can't track per-task usage | High | Yes - new RPC method |
| **Signal Forwarding** | Not implemented | No graceful shutdown | High | Yes - add signal parameter |
| **Per-Task Config** | Defined but unused | All tasks identical | Medium | Yes - extend ExecuteSnippet |
| **Log Streaming** | Polling only | High latency, inefficient | High | Yes - streaming RPC |
| **Status Notifications** | Polling only | Scale issues | Medium | Yes - streaming RPC |
| **Intrinsics Config** | Hardcoded defaults | No validation | Low | Yes - discovery RPC |
| **Execution Timeout** | Not implemented | Runaway tasks | Medium | Yes - add timeout field |
| **Session Persistence** | Unknown | Recovery issues | Low | Clarification needed |

---

## Next Steps

1. **Share this document with Elide team** - Get answers on what's supported
2. **Prioritize based on responses** - Focus on what's feasible in v1.0
3. **Update proto definitions** - Add supported features to API spec
4. **Implement in driver** - Fill in TODOs based on capabilities
5. **Update stubbed server** - Mock new features for continued development

---

## Notes

- This driver is **fully functional with stubbed server** - these are enhancements for production
- All questions are **architectural decisions** for the daemon team, not bugs
- Driver is designed to be **forward-compatible** - can add features without breaking changes
- Session-based architecture (per CTO feedback) is **already implemented** and working

---

**Last Updated**: November 6, 2025
**Status**: Ready for Elide team review

