package runner

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/benedictjohannes/crobe/playbook"
)

func TestRunnerHeadless(t *testing.T) {
	origDisplay := os.Getenv("DISPLAY")
	origWayland := os.Getenv("WAYLAND_DISPLAY")
	_ = os.Unsetenv("DISPLAY")
	_ = os.Unsetenv("WAYLAND_DISPLAY")
	defer func() {
		_ = os.Setenv("DISPLAY", origDisplay)
		_ = os.Setenv("WAYLAND_DISPLAY", origWayland)
	}()

	opts := Options{
		Name:    "crobe-test",
		IsAgent: true,
	}

	// 1. Missing playbook in headless mode
	if code := Run([]string{}, opts); code != 1 {
		t.Errorf("expected exit code 1 for missing playbook, got %d", code)
	}

	// 2. Invalid flag
	if code := Run([]string{"--invalid-flag"}, opts); code != 1 {
		t.Errorf("expected exit code 1 for invalid flag, got %d", code)
	}

	// 3. Custom flag and custom handler
	customHandled := false
	customOpts := Options{
		Name: "crobe-custom",
		CustomFlags: func(fs *flag.FlagSet) {
			fs.Bool("custom", false, "custom flag")
		},
		CustomHandler: func(fs *flag.FlagSet) (bool, int) {
			if f := fs.Lookup("custom"); f != nil && f.Value.String() == "true" {
				customHandled = true
				return true, 42
			}
			return false, 0
		},
	}
	if code := Run([]string{"--custom"}, customOpts); code != 42 || !customHandled {
		t.Errorf("expected exit code 42 for custom handler, got %d", code)
	}

	// 4. Happy path execution
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

	if code := Run([]string{"-folder", tmpDir, pbPath}, opts); code != 0 {
		t.Errorf("expected exit code 0 for happy path, got %d", code)
	}

	// 5. Preprocess hook failure
	hookErrOpts := Options{
		Name:    "crobe-hook",
		IsAgent: false,
		PreprocessHook: func(config *playbook.Playbook, baseDir string) error {
			return errors.New("hook failed")
		},
	}
	if code := Run([]string{"-folder", tmpDir, pbPath}, hookErrOpts); code != 1 {
		t.Errorf("expected exit code 1 for preprocess hook error, got %d", code)
	}

	// 6. Validation error
	valErrPbPath := filepath.Join(tmpDir, "val_err.yaml")
	valErrContent := `
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
	if err := os.WriteFile(valErrPbPath, []byte(valErrContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"-folder", tmpDir, valErrPbPath}, opts); code != 1 {
		t.Errorf("expected exit code 1 for validation error in agent mode, got %d", code)
	}

	// 7. Failing assertion
	failingPbPath := filepath.Join(tmpDir, "failing.yaml")
	failingContent := `
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
	if err := os.WriteFile(failingPbPath, []byte(failingContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"-folder", tmpDir, failingPbPath}, opts); code != 1 {
		t.Errorf("expected exit code 1 for failing assertion, got %d", code)
	}

	// 8. Worker error
	if code := Run([]string{"-worker", "invalid-socket"}, opts); code != 1 {
		t.Errorf("expected exit code 1 for invalid worker socket, got %d", code)
	}

	// 9. Dispatch report network error
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
	if code := Run([]string{"-folder", tmpDir, dispatchErrPbPath}, opts); code != 1 {
		t.Errorf("expected exit code 1 for dispatch network error, got %d", code)
	}

	// 10. Non-existent file
	if code := Run([]string{"-folder", tmpDir, "nonexistent.yaml"}, opts); code != 1 {
		t.Errorf("expected exit code 1 for nonexistent playbook, got %d", code)
	}
}

func TestRunnerUI(t *testing.T) {
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

	opts := Options{
		Name:    "crobe-ui-test",
		IsAgent: true,
	}

	// 1. Test --ui --no-open with preloaded playbook
	exitCodeChan := make(chan int, 1)
	go func() {
		code := Run([]string{"--ui", "--no-open", "--port", "0", pbPath}, opts)
		exitCodeChan <- code
	}()

	time.Sleep(150 * time.Millisecond)
	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)

	select {
	case code := <-exitCodeChan:
		if code != 0 {
			t.Errorf("expected exit code 0 for UI run, got %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for UI server to shut down")
	}

	// 2. Test --ui with invalid preloaded playbook path
	go func() {
		code := Run([]string{"--ui", "--no-open", "--port", "0", filepath.Join(tmpDir, "nonexistent.yaml")}, opts)
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
		code := Run([]string{}, opts)
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
		code := Run([]string{"--ui", "--no-open", "--port", "0", valErrPbPath}, opts)
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
	if code := Run([]string{"--ui", "--host", "999.999.999.999", "--port", "999999"}, opts); code != 1 {
		t.Errorf("expected exit code 1 for invalid host/port, got %d", code)
	}
}
