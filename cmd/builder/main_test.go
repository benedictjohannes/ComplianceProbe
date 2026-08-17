package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestBuilderRun(t *testing.T) {
	// 1. Test missing playbook in headless mode
	origDisplay := os.Getenv("DISPLAY")
	origWayland := os.Getenv("WAYLAND_DISPLAY")
	origDBus := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	_ = os.Unsetenv("DISPLAY")
	_ = os.Unsetenv("WAYLAND_DISPLAY")
	_ = os.Unsetenv("DBUS_SESSION_BUS_ADDRESS")
	defer func() {
		_ = os.Setenv("DISPLAY", origDisplay)
		_ = os.Setenv("WAYLAND_DISPLAY", origWayland)
		_ = os.Setenv("DBUS_SESSION_BUS_ADDRESS", origDBus)
	}()

	if code := run([]string{}); code != 1 {
		t.Errorf("Expected exit code 1 for missing playbook in headless, got %d", code)
	}

	// 2. Test --schema
	if code := run([]string{"--schema"}); code != 0 {
		t.Errorf("Expected exit code 0 for --schema, got %d", code)
	}

	// 3. Test --preprocess (invalid input)
	if code := run([]string{"--preprocess"}); code != 1 {
		t.Errorf("Expected exit code 1 for --preprocess without --input, got %d", code)
	}

	// 4. Test --preprocess (happy path)
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "raw.yaml")
	outputPath := filepath.Join(tmpDir, "baked.yaml")
	inputContent := `
title: Raw
sections:
  - title: Section 1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              script: echo hello
`
	if err := os.WriteFile(inputPath, []byte(inputContent), 0644); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{"--preprocess", "--input", inputPath, "--output", outputPath}); code != 0 {
		t.Errorf("Expected exit code 0 for --preprocess happy path, got %d", code)
	}

	// 5. Test --preprocess with non-existent input file
	if code := run([]string{"--preprocess", "--input", "non-existent.yaml", "--output", outputPath}); code != 1 {
		t.Errorf("Expected exit code 1 for --preprocess with non-existent input, got %d", code)
	}

	// 6. Test --preprocess failure (invalid YAML)
	invalidInputPath := filepath.Join(tmpDir, "invalid_raw.yaml")
	_ = os.WriteFile(invalidInputPath, []byte("invalid: : :"), 0644)
	if code := run([]string{"--preprocess", "--input", invalidInputPath, "--output", outputPath}); code != 1 {
		t.Errorf("Expected exit code 1 for --preprocess with invalid YAML, got %d", code)
	}

	// 7. Test normal run with validation error (duplicate codes)
	invalidPbPath := filepath.Join(tmpDir, "invalid_val.yaml")
	invalidContent := `
title: Invalid
sections:
  - title: Section 1
    description: [D1]
    assertions:
      - code: T1
        title: T1
        cmds: [{exec: {script: echo}}]
      - code: T1
        title: T2
        cmds: [{exec: {script: echo}}]
`
	_ = os.WriteFile(invalidPbPath, []byte(invalidContent), 0644)
	if code := run([]string{"-folder", tmpDir, invalidPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for validation error, got %d", code)
	}

	// 8. Happy path for normal run
	pbPath := filepath.Join(tmpDir, "test_run.yaml")
	pbContent := `
title: Test Run
sections:
  - title: Section 1
    assertions:
      - code: T1
        title: T1
        cmds: [{exec: {script: echo hello}}]
`
	_ = os.WriteFile(pbPath, []byte(pbContent), 0644)
	if code := run([]string{"-folder", tmpDir, pbPath}); code != 0 {
		t.Errorf("Expected exit code 0 for normal run, got %d", code)
	}

	// 9. Test transpilation failure (missing funcFile)
	missingFuncFilePbPath := filepath.Join(tmpDir, "missing_func.yaml")
	missingFuncFileContent := `
title: Missing Func
sections:
  - title: Section 1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              funcFile: missing.js
`
	_ = os.WriteFile(missingFuncFilePbPath, []byte(missingFuncFileContent), 0644)
	if code := run([]string{"-folder", tmpDir, missingFuncFilePbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for missing funcFile, got %d", code)
	}

	// 10. Test custom header flag
	if code := run([]string{"-H", "Auth: token", "--folder", tmpDir, pbPath}); code != 0 {
		t.Errorf("Expected exit code 0 with custom header, got %d", code)
	}

	// 11. Test invalid flag
	if code := run([]string{"--invalid-flag"}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid flag, got %d", code)
	}

	// 12. Test failing assertion
	failingPbPath := filepath.Join(tmpDir, "failing.yaml")
	failingPbContent := `
title: Failing
sections:
  - title: Section 1
    assertions:
      - code: F1
        title: F1
        cmds: [{exec: {script: exit 1}}]
`
	_ = os.WriteFile(failingPbPath, []byte(failingPbContent), 0644)
	if code := run([]string{"-folder", tmpDir, failingPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for failing assertion, got %d", code)
	}

	// 13. Test DispatchReport failure
	dispatchErrPbPath := filepath.Join(tmpDir, "dispatch_err.yaml")
	dispatchErrPbContent := `
title: Dispatch Error
reportDestination: https
sections:
  - title: Section 1
    assertions:
      - code: T1
        title: T1
        cmds: [{exec: {script: echo}}]
`
	_ = os.WriteFile(dispatchErrPbPath, []byte(dispatchErrPbContent), 0644)
	// Do NOT provide -folder flag so it doesn't override to folder
	if code := run([]string{dispatchErrPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for dispatch error, got %d", code)
	}

	// 14. Test LoadConfig failure
	if code := run([]string{"non-existent.yaml"}); code != 1 {
		t.Errorf("Expected exit code 1 for non-existent playbook, got %d", code)
	}

	// 15. Test worker error
	if code := run([]string{"-worker", "invalid-worker-address"}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid worker socket, got %d", code)
	}
}

func TestBuilderUIRun(t *testing.T) {
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
