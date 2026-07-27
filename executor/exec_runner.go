package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ExecutionRequest struct {
	Script    string `json:"script"`
	Shell     string `json:"shell"`
	Extension string `json:"extension"`
}

type ExecCmdRunner interface {
	Run(cmd ExecutionRequest) (ExecutionResult, error)
}

type localExecutionRunnerT struct{}

var localExecutionRunner localExecutionRunnerT
var elevatedExecutionRunner ExecCmdRunner

func SetElevatedExecutionRunner(r ExecCmdRunner) {
	elevatedExecutionRunner = r
}

func GetElevatedExecutionRunner() ExecCmdRunner {
	return elevatedExecutionRunner
}

func (l localExecutionRunnerT) Run(c ExecutionRequest) (ExecutionResult, error) {
	var name string
	var args []string

	tmpDir := os.TempDir()
	var tmpFile string

	shell := c.Shell
	command := c.Script
	extension := c.Extension

	if shell == "" {
		switch runtime.GOOS {
		case "windows":
			if _, err := exec.LookPath("pwsh"); err == nil {
				shell = "pwsh"
			} else {
				shell = "powershell"
			}
		case "darwin":
			shell = "zsh"
		default:
			shell = "bash"
		}
	}

	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	switch shell {
	case "!":
		// Direct execution: command is treated as name + args
		segments := strings.Fields(command)
		if len(segments) == 0 {
			return ExecutionResult{Success: true}, nil
		}
		name = segments[0]
		args = segments[1:]
	case "powershell", "pwsh":
		name = shell
		ext := ".ps1"
		if extension != "" {
			ext = extension
		}
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d%s", time.Now().UnixNano(), ext))
		script := fmt.Sprintf("$ErrorActionPreference = 'Stop'\n%s\n", command)
		os.WriteFile(tmpFile, []byte(script), 0644)
		args = []string{"-ExecutionPolicy", "Bypass", "-File", tmpFile}
	case "bash", "sh", "zsh":
		name = shell
		base := filepath.Base(shell)

		ext := ".sh"
		if extension != "" {
			ext = extension
		} else if base == "zsh" {
			ext = ".zsh"
		}

		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d%s", time.Now().UnixNano(), ext))
		var script string
		if base == "bash" || base == "zsh" {
			script = fmt.Sprintf("set -o pipefail\n%s\n", command)
		} else {
			script = fmt.Sprintf("%s\n", command)
		}
		os.WriteFile(tmpFile, []byte(script), 0755)
		args = []string{tmpFile}
	default:
		// Generic shell/interpreter: shell string is split into command + initial args,
		// and the script is appended as a temporary file.
		shellSegments := strings.Fields(shell)
		name = shellSegments[0]
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d%s", time.Now().UnixNano(), extension))
		os.WriteFile(tmpFile, []byte(command), 0755)

		args = append(shellSegments[1:], tmpFile)
	}

	if tmpFile != "" {
		defer os.Remove(tmpFile)
	}

	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1", "LANG=en_US.UTF-8")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return ExecutionResult{
		Stdout:   CleanupOutput(stdout.String()),
		Stderr:   CleanupOutput(stderr.String()),
		ExitCode: exitCode,
		Success:  err == nil,
	}, nil
}
