# Tests Directory

This directory contains unit tests, integration tests, and test scripts for the Elide task driver.

## Structure

```
tests/
├── unit/              # Unit tests for individual components
│   ├── config_test.go
│   ├── handle_test.go
│   └── state_test.go
├── integration/      # Integration tests with mock daemon
│   └── driver_test.go
├── scripts/          # Test scripts
│   ├── test-end-to-end.sh
│   └── test-integration.sh
└── helpers/          # Test helpers and mocks
    └── mock_daemon_client.go
```

## Running Tests

### Unit Tests

```bash
# Run all unit tests
make test

# Run short unit tests
make test-short

# Run unit tests directly
go test -v ./tests/unit/...
```

### Integration Tests

```bash
# Run integration tests (requires integration tag)
make test-integration

# Run integration tests directly
go test -v -tags=integration ./tests/integration/...
```

### All Tests

```bash
# Run all tests
make test-all
```

### End-to-End Test Script

```bash
# Run end-to-end test script (requires Nomad and stubbed server)
./tests/scripts/test-end-to-end.sh
```

## Test Helpers

### MockDaemonClient

The `MockDaemonClient` in `helpers/mock_daemon_client.go` provides a mock implementation of the `DaemonClient` interface for testing without requiring the actual Elide daemon.

**Features:**
- Mock session management
- Mock execution lifecycle
- Configurable errors for testing error paths
- Simulated execution completion

**Usage:**
```go
mockClient := helpers.NewMockDaemonClient()

// Create session
_, err := mockClient.CreateSession(ctx, "session-1", config)

// Execute snippet
_, err := mockClient.ExecuteSnippet(ctx, "session-1", "exec-1", code, "python", nil, nil)

// Complete execution
mockClient.CompleteExecution("exec-1", 0)
```

## Writing Tests

### Unit Tests

Unit tests should:
- Test individual components in isolation
- Use mocks for external dependencies
- Be fast and deterministic
- Cover both success and error cases

### Integration Tests

Integration tests should:
- Test component interactions
- Use mock daemon client
- Test full workflows
- Be tagged with `// +build integration`

### Test Scripts

Test scripts should:
- Be executable (`chmod +x`)
- Include cleanup logic
- Provide clear output
- Exit with appropriate codes

## Test Coverage

Run with coverage:
```bash
go test -v -cover ./tests/...
```

Generate coverage report:
```bash
go test -v -coverprofile=coverage.out ./tests/...
go tool cover -html=coverage.out
```

