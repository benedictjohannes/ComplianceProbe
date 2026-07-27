package executor

import (
	"os/exec"
	"runtime"
	"testing"

	"github.com/benedictjohannes/crobe/playbook"
)

func wrapRunCmd(script string, shell string, extension string) ExecutionResult {
	res, _ := localExecutionRunner.Run(ExecutionRequest{
		Script:    script,
		Shell:     shell,
		Extension: extension,
	})
	return res
}
func TestRunScript(t *testing.T) {
	// Simple success
	res := wrapRunCmd("echo hello", "", "")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(echo) = %+v; want Success: true, Stdout: hello", res)
	}

	// Failure
	res = wrapRunCmd("ls non_existent_file_12345", "", "")
	if res.Success || res.ExitCode == 0 {
		t.Errorf("RunShell(ls fail) = %+v; want Success: false, ExitCode != 0", res)
	}

	// Default shell (not sh or bash)
	res = wrapRunCmd("echo hello world", "bash", "")
	if !res.Success || res.Stdout != "hello world" {
		t.Errorf("RunShell(bash) = %+v; want Success: true, Stdout: hello world", res)
	}

	if _, err := exec.LookPath("zsh"); err == nil {
		res = wrapRunCmd("echo hello world", "zsh", "")
		if !res.Success || res.Stdout != "hello world" {
			t.Errorf("RunShell(zsh) = %+v; want Success: true, Stdout: hello world", res)
		}
	} else {
		t.Log("Skipping zsh test: zsh not found")
	}

	// Test pipefail for bash/zsh
	// In standard sh, this would succeed (pipefail disabled).
	// In bash/zsh with our setup, it should fail.
	res = wrapRunCmd("non_existent_command | echo hi", "bash", "")
	if res.Success {
		t.Errorf("RunShell(bash pipefail) should have failed, but succeeded: %+v", res)
	}
	if _, err := exec.LookPath("zsh"); err == nil {
		res = wrapRunCmd("non_existent_command | echo hi", "zsh", "")
		if res.Success {
			t.Errorf("RunShell(zsh) pipefail should have failed, but succeeded: %+v", res)
		}
	}

	// Test full path shell
	res = wrapRunCmd("echo full path", "/bin/bash", "")
	if !res.Success || res.Stdout != "full path" {
		t.Errorf("RunShell(/bin/bash) = %+v; want Success: true", res)
	}

	res = wrapRunCmd("echo hello world", "echo", "")
	if !res.Success {
		t.Errorf("RunShell(default) = %+v; want Success: true", res)
	}

	// Non-existent shell (should trigger exit code -1)
	res = wrapRunCmd("echo hello", "/non/existent/shell/cp", "")
	if res.Success || res.ExitCode != -1 {
		t.Errorf("RunShell(missing) = %+v; want Success: false, ExitCode: -1", res)
	}

	// Test case for "!" (direct execution)
	res = wrapRunCmd("echo direct", "!", "")
	if !res.Success || res.Stdout != "direct" {
		t.Errorf("RunShell(!) = %+v; want Success: true, Stdout: direct", res)
	}
	res = wrapRunCmd("", "!", "")
	if !res.Success {
		t.Errorf("RunShell(!) = %+v; want Success: false", res)
	}

	// Test case for custom interpreter logic (e.g., shell "sh -c")
	// Note: using 'sh' because it's available. In practice this could be 'python -u'
	res = wrapRunCmd("echo hello", "sh", ".sh")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(sh with ext) = %+v; want Success: true", res)
	}
	// another try with file extension without dot prefix
	res = wrapRunCmd("echo hello", "sh", "sh")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(sh with ext) = %+v; want Success: true", res)
	}

	// Test sh (hits the 'else' in line 134)
	res = wrapRunCmd("echo hello", "sh", "")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(sh) = %+v", res)
	}

	// Test powershell/pwsh
	res = wrapRunCmd("echo hello", "powershell", "")
	if res.Success || (res.ExitCode != -1 && runtime.GOOS == "linux") {
		t.Logf("RunShell(powershell) failure as expected on linux: %+v", res)
	}
	res = wrapRunCmd("echo hello", "pwsh", "")
	if res.Success || (res.ExitCode != -1 && runtime.GOOS == "linux") {
		t.Logf("RunShell(pwsh) failure as expected on linux: %+v", res)
	}

	// Test default case with generic tool
	res = wrapRunCmd("hello", "cat", "")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(cat) = %+v", res)
	}

	// Test Generic shell with file extension
	res = wrapRunCmd("hello", "cat", ".txt")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(cat .txt) = %+v", res)
	}
}

type mockElevatedRunner struct {
	called bool
	cmd    ExecutionRequest
}

func (m *mockElevatedRunner) Run(cmd ExecutionRequest) (ExecutionResult, error) {
	m.called = true
	m.cmd = cmd
	return ExecutionResult{Stdout: "elevated stdout", Success: true}, nil
}

func TestElevatedRunner(t *testing.T) {
	context := make(map[string]interface{})

	// 1. RequireElevation = true without elevated runner should fail
	eMissing := &playbook.Exec{
		Script:           "echo elevated",
		RequireElevation: true,
	}
	SetElevatedExecutionRunner(nil)
	_, err := RunExec(eMissing, context)
	if err == nil {
		t.Error("RunExec with RequireElevation=true and nil elevatedRunner should return error")
	} else if err.Error() != "elevated runner required but not configured" {
		t.Errorf("Unexpected error message: %v", err)
	}

	// 2. RequireElevation = true with elevated runner registered
	mock := &mockElevatedRunner{}
	SetElevatedExecutionRunner(mock)
	defer SetElevatedExecutionRunner(nil)

	eElevated := &playbook.Exec{
		Script:           "echo elevated",
		Shell:            "bash",
		RequireElevation: true,
	}
	res, err := RunExec(eElevated, context)
	if err != nil {
		t.Fatalf("RunExec elevated error: %v", err)
	}
	if !mock.called {
		t.Error("Mock elevated runner was not called")
	}
	if res.Stdout != "elevated stdout" {
		t.Errorf("Expected 'elevated stdout', got %q", res.Stdout)
	}

	// 3. Getter verification
	if GetElevatedExecutionRunner() != mock {
		t.Errorf("GetElevatedRunner() = %v; want %v", GetElevatedExecutionRunner(), mock)
	}

	// 4. localRunner empty command with shell "!"
	emptyRes, err := localExecutionRunner.Run(ExecutionRequest{Script: "", Shell: "!"})
	if err != nil || !emptyRes.Success {
		t.Errorf("localRunner empty direct command failed: res=%+v, err=%v", emptyRes, err)
	}
}
