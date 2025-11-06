// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	pb "github.com/elide-dev/elide-task-driver/proto/gen/go/elide/daemon/v1alpha1"
)

// Stubbed server implementation for testing
type stubbedServer struct {
	pb.UnimplementedExecutionApiServer

	mu       sync.RWMutex
	sessions map[string]*Session
	executions map[string]*Execution
}

type Session struct {
	ID        string
	Status    pb.SessionStatus
	Config    *pb.SessionConfiguration
	CreatedAt int64
}

type Execution struct {
	ID        string
	SessionID string
	Status    pb.ExecutionStatus
	Complete  bool
	ExitCode  int32
	Stdout    string
	Stderr    string
	Error     string
	CreatedAt time.Time
}

func main() {
	// Default to Unix socket, can override with env var
	socketPath := os.Getenv("ELIDE_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = "/tmp/elide-daemon.sock"
	}

	// Remove existing socket if present
	os.Remove(socketPath)

	// Create Unix socket listener
	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Set socket permissions
	os.Chmod(socketPath, 0666)

	grpcServer := grpc.NewServer()
	pb.RegisterExecutionApiServer(grpcServer, &stubbedServer{
		sessions:   make(map[string]*Session),
		executions: make(map[string]*Execution),
	})

	log.Printf("Stubbed Elide daemon server listening on %s", socketPath)

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down server...")
		grpcServer.GracefulStop()
		os.Remove(socketPath)
		os.Exit(0)
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// CreateSession creates a new session with mocked response
func (s *stubbedServer) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:        req.SessionId,
		Status:    pb.SessionStatus_SESSION_STATUS_ACTIVE,
		Config:    req.Config,
		CreatedAt: time.Now().Unix(),
	}
	s.sessions[req.SessionId] = session

	log.Printf("Created session: %s", req.SessionId)

	return &pb.CreateSessionResponse{
		SessionId: session.ID,
		Status:    session.Status,
		CreatedAt: session.CreatedAt,
	}, nil
}

// GetSession retrieves session information
func (s *stubbedServer) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.GetSessionResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[req.SessionId]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionId)
	}

	return &pb.GetSessionResponse{
		SessionId: session.ID,
		Status:    session.Status,
		Config:    session.Config,
		CreatedAt: session.CreatedAt,
	}, nil
}

// DeleteSession closes a session
func (s *stubbedServer) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.DeleteSessionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[req.SessionId]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionId)
	}

	delete(s.sessions, req.SessionId)
	log.Printf("Deleted session: %s", req.SessionId)

	return &pb.DeleteSessionResponse{Success: true}, nil
}

// ExecuteSnippet executes a snippet with mocked response
func (s *stubbedServer) ExecuteSnippet(ctx context.Context, req *pb.ExecuteSnippetRequest) (*pb.ExecuteSnippetResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify session exists
	_, ok := s.sessions[req.SessionId]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionId)
	}

	// Create execution with mocked status
	exec := &Execution{
		ID:        req.ExecutionId,
		SessionID: req.SessionId,
		Status:    pb.ExecutionStatus_EXECUTION_STATUS_RUNNING,
		Complete:  false,
		CreatedAt: time.Now(),
	}
	s.executions[req.ExecutionId] = exec

	// Simulate async execution completion
	go s.simulateExecution(exec, req.Code, req.Language)

	log.Printf("Started execution: %s in session: %s", req.ExecutionId, req.SessionId)

	return &pb.ExecuteSnippetResponse{
		ExecutionId: exec.ID,
		SessionId:  exec.SessionID,
		Status:     exec.Status,
	}, nil
}

// simulateExecution simulates snippet execution with mocked results
func (s *stubbedServer) simulateExecution(exec *Execution, code string, language string) {
	// Simulate execution time
	time.Sleep(2 * time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Mock successful execution
	exec.Status = pb.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	exec.Complete = true
	exec.ExitCode = 0
	exec.Stdout = fmt.Sprintf("Mocked output for %s snippet:\n%s", language, code)
	exec.Stderr = ""
}

// GetExecutionStatus retrieves execution status
func (s *stubbedServer) GetExecutionStatus(ctx context.Context, req *pb.GetExecutionStatusRequest) (*pb.GetExecutionStatusResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exec, ok := s.executions[req.ExecutionId]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", req.ExecutionId)
	}

	return &pb.GetExecutionStatusResponse{
		ExecutionId: exec.ID,
		SessionId:   exec.SessionID,
		Status:     exec.Status,
		Complete:   exec.Complete,
		ExitCode:   exec.ExitCode,
		Stdout:     exec.Stdout,
		Stderr:     exec.Stderr,
		Error:      exec.Error,
	}, nil
}

// CancelExecution cancels a running execution
func (s *stubbedServer) CancelExecution(ctx context.Context, req *pb.CancelExecutionRequest) (*pb.CancelExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, ok := s.executions[req.ExecutionId]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", req.ExecutionId)
	}

	if exec.Complete {
		return &pb.CancelExecutionResponse{Success: false}, nil
	}

	exec.Status = pb.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	exec.Complete = true
	exec.ExitCode = -1

	log.Printf("Cancelled execution: %s", req.ExecutionId)

	return &pb.CancelExecutionResponse{Success: true}, nil
}

// Health checks daemon health
func (s *stubbedServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Healthy: true,
		Version: "stubbed-v0.1.0",
	}, nil
}

