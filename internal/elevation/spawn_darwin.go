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

	// Strategy 1: Non-interactive sudo (fast path for passwordless sudo / cached credentials).
	// Explicitly cmd.Run() sudo credential probing using 'sudo -n -v' know privilege immediately.
	// Using cmd.Start() directly on 'sudo -n' produces a false positive.
	if err := exec.Command("sudo", "-n", "-v").Run(); err == nil {
		cmd := exec.Command("sudo", "-n", execPath, "--worker", socketURI)
		if err := cmd.Start(); err == nil {
			return &ProcessHandle{cmd: cmd}, nil
		}
	}

	// Strategy 2: macOS GUI elevation prompt via osascript
	// shell script: Shell-escape execPath and socketURI,
	// redirecting to /dev/null and backgrounding (&)
	innerShellCmd := fmt.Sprintf("%s --worker %s > /dev/null 2>&1 &", strconv.Quote(execPath), strconv.Quote(socketURI))
	// quote to AppleScript literal syntax
	// executing the inner script blocks, hence detaching and backgrounding
	script := fmt.Sprintf("do shell script %s with administrator privileges", strconv.Quote(innerShellCmd))
	cmdOsa := exec.Command("osascript", "-e", script)
	cmdOsa.Stderr = os.Stderr // Pipe osascript errors to stderr for diagnostic visibility
	// Use cmdOsa.Run() instead of cmdOsa.Start() to wait for the user to approve/cancel the dialog.
	// If cancelled or blocked by TCC, it returns an error and cleanly falls through to Strategy 3.
	if err := cmdOsa.Run(); err == nil {
		// Worker is running detached as root in the background.
		// Lifecycle is managed via the domain socket (e.g. sending a shutdown command).
		return &ProcessHandle{cmd: nil}, nil
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
	// Credentials are now active in the cache; spawn the background worker non-interactively.
	cmdWorker := exec.Command("sudo", "-n", execPath, "--worker", socketURI)
	if err := cmdWorker.Start(); err != nil {
		return nil, fmt.Errorf("failed to start elevated worker after authentication: %w", err)
	}

	return &ProcessHandle{cmd: cmdWorker}, nil
}
