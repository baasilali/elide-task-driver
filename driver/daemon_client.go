// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/elide-dev/elide-task-driver/proto/gen/go/elide/daemon/v1alpha1"
)

// DaemonClient is the interface for communicating with the Elide daemon
type DaemonClient interface {
	// Session Management
	CreateSession(ctx context.Context, sessionID string, config *pb.SessionConfiguration) (*pb.CreateSessionResponse, error)
	GetSession(ctx context.Context, sessionID string) (*pb.GetSessionResponse, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Execution within Session
	ExecuteSnippet(ctx context.Context, sessionID string, executionID string, code string, language string, env map[string]string, args []string) (*pb.ExecuteSnippetResponse, error)
	GetExecutionStatus(ctx context.Context, sessionID string, executionID string) (*pb.GetExecutionStatusResponse, error)
	CancelExecution(ctx context.Context, sessionID string, executionID string) error

	// Health check
	Health(ctx context.Context) error

	// Close closes the connection to the daemon
	Close() error
}

// elideDaemonClient is the implementation of DaemonClient
type elideDaemonClient struct {
	conn            *grpc.ClientConn
	executionClient pb.ExecutionApiClient
	sessionID       string // Cached session ID for this client
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
		conn:            conn,
		executionClient: pb.NewExecutionApiClient(conn),
	}, nil
}

// CreateSession creates a new session with the given configuration
func (c *elideDaemonClient) CreateSession(ctx context.Context, sessionID string, config *pb.SessionConfiguration) (*pb.CreateSessionResponse, error) {
	resp, err := c.executionClient.CreateSession(ctx, &pb.CreateSessionRequest{
		SessionId: sessionID,
		Config:    config,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	c.sessionID = sessionID
	return resp, nil
}

// GetSession retrieves session information
func (c *elideDaemonClient) GetSession(ctx context.Context, sessionID string) (*pb.GetSessionResponse, error) {
	resp, err := c.executionClient.GetSession(ctx, &pb.GetSessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}
	return resp, nil
}

// DeleteSession closes and cleans up a session
func (c *elideDaemonClient) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := c.executionClient.DeleteSession(ctx, &pb.DeleteSessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete session: %v", err)
	}
	if c.sessionID == sessionID {
		c.sessionID = ""
	}
	return nil
}

// ExecuteSnippet executes a code snippet within a session
func (c *elideDaemonClient) ExecuteSnippet(ctx context.Context, sessionID string, executionID string, code string, language string, env map[string]string, args []string) (*pb.ExecuteSnippetResponse, error) {
	resp, err := c.executionClient.ExecuteSnippet(ctx, &pb.ExecuteSnippetRequest{
		SessionId:   sessionID,
		ExecutionId: executionID,
		Code:        code,
		Language:    language,
		Env:         env,
		Args:        args,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute snippet: %v", err)
	}
	return resp, nil
}

// GetExecutionStatus gets the current status of an execution
func (c *elideDaemonClient) GetExecutionStatus(ctx context.Context, sessionID string, executionID string) (*pb.GetExecutionStatusResponse, error) {
	resp, err := c.executionClient.GetExecutionStatus(ctx, &pb.GetExecutionStatusRequest{
		SessionId:   sessionID,
		ExecutionId: executionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get execution status: %v", err)
	}
	return resp, nil
}

// CancelExecution cancels a running execution
func (c *elideDaemonClient) CancelExecution(ctx context.Context, sessionID string, executionID string) error {
	_, err := c.executionClient.CancelExecution(ctx, &pb.CancelExecutionRequest{
		SessionId:   sessionID,
		ExecutionId: executionID,
	})
	if err != nil {
		return fmt.Errorf("failed to cancel execution: %v", err)
	}
	return nil
}

// Health checks if the daemon is healthy
func (c *elideDaemonClient) Health(ctx context.Context) error {
	_, err := c.executionClient.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}
	return nil
}

// Close closes the connection to the daemon
func (c *elideDaemonClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
