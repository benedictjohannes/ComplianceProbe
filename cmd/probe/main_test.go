package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestProbeRun(t *testing.T) {
	// 1. Test missing playbook in headless mode
	origDisplay := os.Getenv("DISPLAY")
	origWayland := os.Getenv("WAYLAND_DISPLAY")
	_ = os.Unsetenv("DISPLAY")
	_ = os.Unsetenv("WAYLAND_DISPLAY")
	defer func() {
		_ = os.Setenv("DISPLAY", origDisplay)
		_ = os.Setenv("WAYLAND_DISPLAY", origWayland)
	}()

	if code := run([]string{}); code != 1 {
		t.Errorf("Expected exit code 1 for missing playbook in headless, got %d", code)
	}

	// 2. Test invalid flag
	if code := run([]string{"--invalid-flag"}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid flag, got %d", code)
	}

	// 3. Happy path (using a minimal playbook)
	tmpDir := t.TempDir()
	pbPath := filepath.Join(tmpDir, "test.yaml")
	pbContent := `
title: Test
sections:
  - title: Section 1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              script: echo hello
`
	if err := os.WriteFile(pbPath, []byte(pbContent), 0644); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{"-folder", tmpDir, pbPath}); code != 0 {
		t.Errorf("Expected exit code 0 for happy path, got %d", code)
	}

	// 4. Test missing playbook file
	if code := run([]string{"-folder", tmpDir, "non-existent.yaml"}); code != 1 {
		t.Errorf("Expected exit code 1 for non-existent playbook, got %d", code)
	}

	// 5. Test invalid playbook content (YAML error)
	invalidPbPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidPbPath, []byte("invalid: yaml: :"), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-folder", tmpDir, invalidPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid playbook content, got %d", code)
	}

	// 6. Test validation error (e.g. funcFile in agent mode)
	validationErrPbPath := filepath.Join(tmpDir, "validation_err.yaml")
	validationErrPbContent := `
title: Test
sections:
  - title: Section 1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              funcFile: some-script.js
`
	if err := os.WriteFile(validationErrPbPath, []byte(validationErrPbContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-folder", tmpDir, validationErrPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for validation error, got %d", code)
	}

	// 7. Test playbook with failing assertion
	failingPbPath := filepath.Join(tmpDir, "failing.yaml")
	failingPbContent := `
title: Test
sections:
  - title: Section 1
    assertions:
      - code: F1
        title: F1
        cmds:
          - exec:
              script: exit 1
`
	if err := os.WriteFile(failingPbPath, []byte(failingPbContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-folder", tmpDir, failingPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for failing assertion, got %d", code)
	}

	// 8. Test worker error
	if code := run([]string{"-worker", "invalid-worker-address"}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid worker socket, got %d", code)
	}

	// 9. Test DispatchReport network failure (valid config, but endpoint unreachable)
	dispatchErrPbPath := filepath.Join(tmpDir, "dispatch_err.yaml")
	dispatchErrPbContent := `
title: Dispatch Error
reportDestination: https
destinationConfig:
  url: "http://127.0.0.1:1/nonexistent"
sections:
  - title: Section 1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              script: echo hello
`
	if err := os.WriteFile(dispatchErrPbPath, []byte(dispatchErrPbContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-folder", tmpDir, dispatchErrPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for dispatch network error, got %d", code)
	}
	// 10. Test headless with custom header flag
	if code := run([]string{"-H", "Authorization: Bearer test", "-folder", tmpDir, pbPath}); code != 0 {
		t.Errorf("Expected exit code 0 for custom header flag, got %d", code)
	}
}

func TestProbeUIRun(t *testing.T) {
	tmpDir := t.TempDir()
	pbPath := filepath.Join(tmpDir, "preloaded.yaml")
	pbContent := `
title: Preloaded Test
sections:
  - title: Section 1
    assertions:
      - code: P1
        title: P1
        cmds:
          - exec:
              script: echo preloaded
`
	if err := os.WriteFile(pbPath, []byte(pbContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Test --ui --no-open with preloaded playbook
	exitCodeChan := make(chan int, 1)
	go func() {
		code := run([]string{"--ui", "--no-open", "--port", "0", pbPath})
		exitCodeChan <- code
	}()

	// Allow server to spin up and bind
	time.Sleep(150 * time.Millisecond)

	// Send interrupt signal to gracefully shut down server
	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)

	select {
	case code := <-exitCodeChan:
		if code != 0 {
			t.Errorf("expected exit code 0 for UI run, got %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for UI server to shut down")
	}

	// 2. Test --ui with invalid preloaded playbook path (sets load error and starts UI)
	go func() {
		code := run([]string{"--ui", "--no-open", "--port", "0", filepath.Join(tmpDir, "nonexistent.yaml")})
		exitCodeChan <- code
	}()
	time.Sleep(150 * time.Millisecond)
	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
	select {
	case code := <-exitCodeChan:
		if code != 0 {
			t.Errorf("expected exit code 0 for UI run with bad playbook, got %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for UI server to shut down")
	}

	// 3. Test PORT env var and zero-arg desktop auto launch
	origDisplay := os.Getenv("DISPLAY")
	origPort := os.Getenv("PORT")
	defer func() {
		_ = os.Setenv("DISPLAY", origDisplay)
		_ = os.Setenv("PORT", origPort)
	}()

	_ = os.Setenv("DISPLAY", ":0")
	_ = os.Setenv("PORT", "0")

	go func() {
		code := run([]string{})
		exitCodeChan <- code
	}()

	time.Sleep(150 * time.Millisecond)
	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
	select {
	case code := <-exitCodeChan:
		if code != 0 {
			t.Errorf("expected exit code 0 for zero-arg desktop auto launch, got %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for UI server to shut down")
	}

	// 4. Test preloaded playbook with validation error in UI mode
	valErrPbPath := filepath.Join(tmpDir, "preloaded_val_err.yaml")
	valErrContent := `
title: "Duplicate codes"
sections:
  - title: "Section 1"
    assertions:
      - code: "DUP-001"
        title: "Assertion 1"
        cmds:
          - exec:
              script: "echo 1"
      - code: "DUP-001"
        title: "Assertion 2"
        cmds:
          - exec:
              script: "echo 2"
`
	_ = os.WriteFile(valErrPbPath, []byte(valErrContent), 0644)
	go func() {
		code := run([]string{"--ui", "--no-open", "--port", "0", valErrPbPath})
		exitCodeChan <- code
	}()
	time.Sleep(150 * time.Millisecond)
	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
	select {
	case code := <-exitCodeChan:
		if code != 0 {
			t.Errorf("expected exit code 0 for UI run with validation error playbook, got %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for UI server to shut down")
	}

	// 5. Test UI start failure (invalid host)
	if code := run([]string{"--ui", "--host", "999.999.999.999", "--port", "999999"}); code != 1 {
		t.Errorf("expected exit code 1 for invalid host/port, got %d", code)
	}
}



