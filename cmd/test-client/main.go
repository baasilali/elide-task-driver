// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/elide-dev/elide-task-driver/proto/gen/go/elide/daemon/v1alpha1"
)

func main() {
	// Connect to stubbed server
	socketPath := "/tmp/elide-daemon.sock"
	
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		return net.Dial("unix", addr)
	}

	conn, err := grpc.Dial(
		socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewExecutionApiClient(conn)

	// Test Health
	fmt.Println("Testing Health...")
	healthResp, err := client.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("✓ Health: %v (version: %s)\n\n", healthResp.Healthy, healthResp.Version)

	// Test CreateSession
	fmt.Println("Testing CreateSession...")
	sessionID := fmt.Sprintf("test-session-%d", time.Now().Unix())
	createResp, err := client.CreateSession(context.Background(), &pb.CreateSessionRequest{
		SessionId: sessionID,
		Config: &pb.SessionConfiguration{
			ContextPoolSize: 10,
			EnabledLanguages: []string{"python", "javascript"},
			EnabledIntrinsics: []string{"io", "env"},
			MemoryLimitMb: 512,
			EnableAi: false,
		},
	})
	if err != nil {
		log.Fatalf("CreateSession failed: %v", err)
	}
	fmt.Printf("✓ Session created: %s (status: %s)\n\n", createResp.SessionId, createResp.Status)

	// Test ExecuteSnippet
	fmt.Println("Testing ExecuteSnippet...")
	executionID := fmt.Sprintf("test-exec-%d", time.Now().Unix())
	execResp, err := client.ExecuteSnippet(context.Background(), &pb.ExecuteSnippetRequest{
		SessionId:   sessionID,
		ExecutionId: executionID,
		Code:        "print('Hello from Elide!')",
		Language:    "python",
		Env:         map[string]string{"TEST": "value"},
		Args:        []string{},
	})
	if err != nil {
		log.Fatalf("ExecuteSnippet failed: %v", err)
	}
	fmt.Printf("✓ Execution started: %s (status: %s)\n\n", execResp.ExecutionId, execResp.Status)

	// Wait a bit for execution to complete
	fmt.Println("Waiting for execution to complete...")
	time.Sleep(3 * time.Second)

	// Test GetExecutionStatus
	fmt.Println("Testing GetExecutionStatus...")
	statusResp, err := client.GetExecutionStatus(context.Background(), &pb.GetExecutionStatusRequest{
		SessionId:   sessionID,
		ExecutionId: executionID,
	})
	if err != nil {
		log.Fatalf("GetExecutionStatus failed: %v", err)
	}
	fmt.Printf("✓ Execution status: %s (complete: %v, exit_code: %d)\n", statusResp.Status, statusResp.Complete, statusResp.ExitCode)
	if statusResp.Stdout != "" {
		fmt.Printf("  stdout: %s\n", statusResp.Stdout)
	}

	// Test DeleteSession
	fmt.Println("\nTesting DeleteSession...")
	deleteResp, err := client.DeleteSession(context.Background(), &pb.DeleteSessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		log.Fatalf("DeleteSession failed: %v", err)
	}
	fmt.Printf("✓ Session deleted: %v\n", deleteResp.Success)

	fmt.Println("\n✅ All tests passed!")
}

