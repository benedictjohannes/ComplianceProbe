//go:build linux

package elevation

import (
	"fmt"
	"os"
	"os/exec"
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

	// Strategy 2: GUI Polkit prompt via pkexec
	if _, err := exec.LookPath("pkexec"); err == nil {
		cmdPkexec := exec.Command("pkexec", execPath, "--worker", socketURI)
		if err := cmdPkexec.Start(); err == nil {
			return &ProcessHandle{cmd: cmdPkexec}, nil
		}
	}

	// Strategy 3: Interactive terminal sudo
	cmdInteractive := exec.Command("sudo", execPath, "--worker", socketURI)
	cmdInteractive.Stdin = os.Stdin
	cmdInteractive.Stdout = os.Stdout
	cmdInteractive.Stderr = os.Stderr

	if err := cmdInteractive.Start(); err != nil {
		return nil, fmt.Errorf("failed to spawn elevated worker via sudo (non-interactive, pkexec, and interactive failed): %w", err)
	}

	return &ProcessHandle{cmd: cmdInteractive}, nil
}
