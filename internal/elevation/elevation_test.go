package elevation

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
)

func TestProtocolSerialization(t *testing.T) {
	msg := WireMessage{
		Type: MsgTypeExec,
		ID:   "test-id-123",
		Exec: &executor.ExecutionRequest{
			Script: "echo test",
			Shell:  "bash",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded WireMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Type != MsgTypeExec || decoded.ID != "test-id-123" || decoded.Exec.Script != "echo test" {
		t.Errorf("Decoded message mismatch: %+v", decoded)
	}
}

func TestWorkerClientIPC(t *testing.T) {
	socketURI := GenerateSocketURI()
	listener, err := Listen(socketURI)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()
	defer CleanupListener(socketURI)

	// Run worker in background goroutine
	workerErrChan := make(chan error, 1)
	go func() {
		workerErrChan <- RunWorker(socketURI)
	}()

	// Accept worker connection on listener
	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("Accept error: %v", err)
	}

	client := NewClientWithConn(conn, listener, socketURI, nil)

	// Test Ping / Pong
	if err := client.Ping(); err != nil {
		t.Errorf("Client.Ping failed: %v", err)
	}

	// Test Execution Request over IPC
	res, err := client.Run(context.Background(), executor.ExecutionRequest{
		Script: "echo IPC_SUCCESS",
		Shell:  "bash",
	})
	if err != nil {
		t.Fatalf("Client.Run failed: %v", err)
	}

	if !res.Success || res.Stdout != "IPC_SUCCESS" {
		t.Errorf("Unexpected execution result: %+v", res)
	}

	// Test Close (Shutdown signal)
	if err := client.Close(); err != nil {
		t.Errorf("Client.Close failed: %v", err)
	}

	select {
	case wErr := <-workerErrChan:
		if wErr != nil {
			t.Errorf("Worker exited with error: %v", wErr)
		}
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting for worker goroutine to terminate")
	}
}

func TestWorkerShutdownCancelsRunningProcess(t *testing.T) {
	socketURI := GenerateSocketURI()
	listener, err := Listen(socketURI)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()
	defer CleanupListener(socketURI)

	workerErrChan := make(chan error, 1)
	go func() {
		workerErrChan <- RunWorker(socketURI)
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("Accept error: %v", err)
	}

	client := NewClientWithConn(conn, listener, socketURI, nil)

	// Launch long running process on worker in background
	go func() {
		_, _ = client.Run(context.Background(), executor.ExecutionRequest{
			Script: "sleep 10",
			Shell:  "bash",
		})
	}()

	time.Sleep(100 * time.Millisecond)

	// Close client sending shutdown signal
	closeStart := time.Now()
	_ = client.Close()

	select {
	case wErr := <-workerErrChan:
		if wErr != nil {
			t.Errorf("Worker exited with error: %v", wErr)
		}
		if time.Since(closeStart) > 2*time.Second {
			t.Errorf("Worker shutdown took %v; expected immediate process cancellation", time.Since(closeStart))
		}
	case <-time.After(3 * time.Second):
		t.Error("Worker failed to shut down when long-running command was active")
	}
}

type dummyRunner struct{}

func (d dummyRunner) Run(ctx context.Context, cmd executor.ExecutionRequest) (executor.ExecutionResult, error) {
	return executor.ExecutionResult{Success: true}, nil
}

func TestSetupElevation(t *testing.T) {
	// Ensure executor runner state is restored after tests
	origRunner := executor.GetElevatedExecutionRunner()
	defer executor.SetElevatedExecutionRunner(origRunner)

	t.Run("nil playbook returns noop cleanup and no error", func(t *testing.T) {
		executor.SetElevatedExecutionRunner(nil)
		cleanup, err := SetupElevation(nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cleanup == nil {
			t.Fatal("expected non-nil cleanup func")
		}
		cleanup()
		if executor.GetElevatedExecutionRunner() != nil {
			t.Error("expected elevated runner to remain nil")
		}
	})

	t.Run("playbook without elevation required returns noop cleanup", func(t *testing.T) {
		executor.SetElevatedExecutionRunner(nil)
		pb := &playbook.Playbook{
			Sections: []playbook.Section{
				{
					Assertions: []playbook.Assertion{
						{
							Cmds: []playbook.Cmd{
								{Exec: playbook.Exec{Script: "echo hello", RequireElevation: false}},
							},
						},
					},
				},
			},
		}

		cleanup, err := SetupElevation(pb)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cleanup == nil {
			t.Fatal("expected non-nil cleanup func")
		}
		cleanup()
		if executor.GetElevatedExecutionRunner() != nil {
			t.Error("expected elevated runner to remain nil")
		}
	})

	t.Run("runner already configured is preserved and returns noop cleanup", func(t *testing.T) {
		dummy := dummyRunner{}
		executor.SetElevatedExecutionRunner(dummy)

		pb := &playbook.Playbook{
			Sections: []playbook.Section{
				{
					Assertions: []playbook.Assertion{
						{
							Cmds: []playbook.Cmd{
								{Exec: playbook.Exec{Script: "sudo whoami", RequireElevation: true}},
							},
						},
					},
				},
			},
		}

		cleanup, err := SetupElevation(pb)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cleanup == nil {
			t.Fatal("expected non-nil cleanup func")
		}
		cleanup()

		if executor.GetElevatedExecutionRunner() != dummy {
			t.Error("expected existing runner to be preserved")
		}
	})
}
