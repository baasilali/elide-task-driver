// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DaemonClient is the interface for communicating with the Elide daemon
// This will be implemented once the daemon API is available
type DaemonClient interface {
	// ExecuteSnippet executes a code snippet in the Elide daemon
	// Returns execution ID and initial status
	ExecuteSnippet(ctx context.Context, req *ExecuteSnippetRequest) (*ExecuteSnippetResponse, error)

	// GetExecutionStatus gets the current status of an execution
	GetExecutionStatus(ctx context.Context, executionId string) (*ExecutionStatus, error)

	// CancelExecution cancels a running execution
	CancelExecution(ctx context.Context, executionId string) error

	// Health checks if the daemon is healthy
	Health(ctx context.Context) error

	// Close closes the connection to the daemon
	Close() error
}

// ExecuteSnippetRequest represents a request to execute a code snippet
// TODO: This will be replaced by generated proto types once proto definitions are available
type ExecuteSnippetRequest struct {
	Code        string            // Code snippet to execute
	Language    string            // Language: python, javascript, typescript
	Env         map[string]string // Environment variables
	ExecutionId string            // Unique execution ID (typically task ID)
	Args        []string          // Arguments to pass to script
}

// ExecuteSnippetResponse represents the response from ExecuteSnippet
// TODO: This will be replaced by generated proto types once proto definitions are available
type ExecuteSnippetResponse struct {
	ExecutionId string // Execution ID (may differ from request if daemon generates one)
	Status      string // Initial status: running, queued, etc.
}

// ExecutionStatus represents the current status of an execution
// TODO: This will be replaced by generated proto types once proto definitions are available
type ExecutionStatus struct {
	ExecutionId string // Execution ID
	Status      string // Current status: running, completed, failed, cancelled
	Complete    bool   // Whether execution is complete
	ExitCode    int    // Exit code (if completed)
	Stdout      string // Standard output (if available)
	Stderr      string // Standard error (if available)
	Error       string // Error message (if failed)
}

// elideDaemonClient is the implementation of DaemonClient
// This will use gRPC to communicate with the Elide daemon
type elideDaemonClient struct {
	conn *grpc.ClientConn
	// TODO: Add gRPC client once proto stubs are generated
	// executionClient executionapi.ExecutionApiClient
}

// NewDaemonClient creates a new client connected to the Elide daemon
// It supports both Unix socket and TCP connections
func NewDaemonClient(socketPath string, tcpAddress string) (DaemonClient, error) {
	var conn *grpc.ClientConn
	var err error

	if socketPath != "" {
		// Connect via Unix socket
		dialer := func(ctx context.Context, addr string) (net.Conn, error) {
			return net.Dial("unix", addr)
		}

		conn, err = grpc.Dial(
			socketPath,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(dialer),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect via Unix socket: %v", err)
		}
	} else if tcpAddress != "" {
		// Connect via TCP
		conn, err = grpc.Dial(
			tcpAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect via TCP: %v", err)
		}
	} else {
		return nil, fmt.Errorf("either socket_path or tcp_address must be specified")
	}

	return &elideDaemonClient{
		conn: conn,
	}, nil
}

// ExecuteSnippet executes a code snippet in the Elide daemon
// TODO: Implement once proto stubs are available
func (c *elideDaemonClient) ExecuteSnippet(ctx context.Context, req *ExecuteSnippetRequest) (*ExecuteSnippetResponse, error) {
	// TODO: Replace with actual gRPC call once proto stubs are generated
	// Example:
	//   resp, err := c.executionClient.ExecuteSnippet(ctx, &executionapi.ExecuteSnippetRequest{
	//       Code:        req.Code,
	//       Language:    req.Language,
	//       Env:         req.Env,
	//       ExecutionId: req.ExecutionId,
	//       Args:        req.Args,
	//   })
	//   if err != nil {
	//       return nil, err
	//   }
	//   return &ExecuteSnippetResponse{
	//       ExecutionId: resp.ExecutionId,
	//       Status:      resp.Status.String(),
	//   }, nil

	return nil, fmt.Errorf("daemon API not yet available - proto stubs not generated")
}

// GetExecutionStatus gets the current status of an execution
// TODO: Implement once proto stubs are available
func (c *elideDaemonClient) GetExecutionStatus(ctx context.Context, executionId string) (*ExecutionStatus, error) {
	// TODO: Replace with actual gRPC call once proto stubs are generated
	// Example:
	//   resp, err := c.executionClient.GetExecutionStatus(ctx, &executionapi.GetExecutionStatusRequest{
	//       ExecutionId: executionId,
	//   })
	//   if err != nil {
	//       return nil, err
	//   }
	//   return &ExecutionStatus{
	//       ExecutionId: resp.ExecutionId,
	//       Status:      resp.Status.String(),
	//       Complete:    resp.Complete,
	//       ExitCode:    int(resp.ExitCode),
	//       Stdout:      resp.Stdout,
	//       Stderr:      resp.Stderr,
	//       Error:       resp.Error,
	//   }, nil

	return nil, fmt.Errorf("daemon API not yet available - proto stubs not generated")
}

// CancelExecution cancels a running execution
// TODO: Implement once proto stubs are available
func (c *elideDaemonClient) CancelExecution(ctx context.Context, executionId string) error {
	// TODO: Replace with actual gRPC call once proto stubs are generated
	// Example:
	//   _, err := c.executionClient.CancelExecution(ctx, &executionapi.CancelExecutionRequest{
	//       ExecutionId: executionId,
	//   })
	//   return err

	return fmt.Errorf("daemon API not yet available - proto stubs not generated")
}

// Health checks if the daemon is healthy
// TODO: Implement once proto stubs are available
func (c *elideDaemonClient) Health(ctx context.Context) error {
	// TODO: Replace with actual gRPC call once proto stubs are generated
	// Example:
	//   _, err := c.executionClient.Health(ctx, &executionapi.HealthRequest{})
	//   return err

	return fmt.Errorf("daemon API not yet available - proto stubs not generated")
}

// Close closes the connection to the daemon
func (c *elideDaemonClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

