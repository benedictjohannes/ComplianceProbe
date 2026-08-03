//go:build darwin

package elevation

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

type ProcessHandle struct {
	cmd *exec.Cmd
}

func (p *ProcessHandle) Wait() error {
	if p == nil || p.cmd == nil {
		return nil
	}
	return p.cmd.Wait()
}

func (p *ProcessHandle) Kill() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func SpawnWorker(socketURI string) (*ProcessHandle, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to locate executable: %w", err)
	}

	// Strategy 1: Non-interactive sudo (fast path for passwordless sudo / cached credentials)
	cmd := exec.Command("sudo", "-n", execPath, "--worker", socketURI)
	if err := cmd.Start(); err == nil {
		return &ProcessHandle{cmd: cmd}, nil
	}

	// Strategy 2: macOS GUI elevation prompt via osascript
	script := fmt.Sprintf("do shell script %s with administrator privileges", strconv.Quote(fmt.Sprintf("%s --worker %s", execPath, socketURI)))
	cmdOsa := exec.Command("osascript", "-e", script)
	if err := cmdOsa.Start(); err == nil {
		return &ProcessHandle{cmd: cmdOsa}, nil
	}

	// Strategy 3: Interactive terminal sudo
	cmdInteractive := exec.Command("sudo", execPath, "--worker", socketURI)
	cmdInteractive.Stdin = os.Stdin
	cmdInteractive.Stdout = os.Stdout
	cmdInteractive.Stderr = os.Stderr

	if err := cmdInteractive.Start(); err != nil {
		return nil, fmt.Errorf("failed to spawn elevated worker via sudo (non-interactive, osascript, and interactive failed): %w", err)
	}

	return &ProcessHandle{cmd: cmdInteractive}, nil
}
