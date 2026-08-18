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

	// Strategy 1: Non-interactive sudo (fast path for passwordless sudo / cached credentials).
	// Explicitly cmd.Run() sudo credential probing using 'sudo -n -v' know privilege immediately.
	// Using cmd.Start() directly on 'sudo -n' produces a false positive.
	if err := exec.Command("sudo", "-n", "-v").Run(); err == nil {
		cmd := exec.Command("sudo", "-n", execPath, "--worker", socketURI)
		if err := cmd.Start(); err == nil {
			return &ProcessHandle{cmd: cmd}, nil
		}
	}

	// Strategy 2: GUI Polkit prompt via pkexec
	// pkexec directly manages the worker child without an intermediate shell wrapper.
	if _, err := exec.LookPath("pkexec"); err == nil {
		cmdPkexec := exec.Command("pkexec", execPath, "--worker", socketURI)
		if err := cmdPkexec.Start(); err == nil {
			return &ProcessHandle{cmd: cmdPkexec}, nil
		}
	}

	// Strategy 3: Interactive terminal sudo fallback
	// synchronous interactive sudo prompt, approvals of which will be cached
	cmdAuth := exec.Command("sudo", "-v")
	cmdAuth.Stdin = os.Stdin
	cmdAuth.Stdout = os.Stdout
	cmdAuth.Stderr = os.Stderr
	if err := cmdAuth.Run(); err != nil {
		return nil, fmt.Errorf("failed to spawn elevated worker (all elevation strategies failed): %w", err)
	}
	// Credentials are now cached in the session; launch the worker non-interactively.
	cmdWorker := exec.Command("sudo", "-n", execPath, "--worker", socketURI)
	if err := cmdWorker.Start(); err != nil {
		return nil, fmt.Errorf("failed to start elevated worker after authentication: %w", err)
	}

	return &ProcessHandle{cmd: cmdWorker}, nil
}
