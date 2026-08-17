package desktop

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)


func TestIsDesktopGUI(t *testing.T) {
	switch runtime.GOOS {
	case "linux":
		// Test when DISPLAY / WAYLAND_DISPLAY are unset
		origDisplay := os.Getenv("DISPLAY")
		origWayland := os.Getenv("WAYLAND_DISPLAY")
		defer func() {
			_ = os.Setenv("DISPLAY", origDisplay)
			_ = os.Setenv("WAYLAND_DISPLAY", origWayland)
		}()

		_ = os.Unsetenv("DISPLAY")
		_ = os.Unsetenv("WAYLAND_DISPLAY")
		if IsDesktopGUI() {
			t.Errorf("expected IsDesktopGUI() to be false when DISPLAY and WAYLAND_DISPLAY are unset")
		}

		_ = os.Setenv("DISPLAY", ":0")
		if !IsDesktopGUI() {
			t.Errorf("expected IsDesktopGUI() to be true when DISPLAY is set")
		}

		_ = os.Unsetenv("DISPLAY")
		_ = os.Setenv("WAYLAND_DISPLAY", "wayland-0")
		if !IsDesktopGUI() {
			t.Errorf("expected IsDesktopGUI() to be true when WAYLAND_DISPLAY is set")
		}

	case "darwin":
		origSSH1 := os.Getenv("SSH_CONNECTION")
		origSSH2 := os.Getenv("SSH_CLIENT")
		origSSH3 := os.Getenv("SSH_TTY")
		defer func() {
			_ = os.Setenv("SSH_CONNECTION", origSSH1)
			_ = os.Setenv("SSH_CLIENT", origSSH2)
			_ = os.Setenv("SSH_TTY", origSSH3)
		}()

		_ = os.Unsetenv("SSH_CONNECTION")
		_ = os.Unsetenv("SSH_CLIENT")
		_ = os.Unsetenv("SSH_TTY")
		if !IsDesktopGUI() {
			t.Errorf("expected IsDesktopGUI() to be true when no SSH environment variables are set")
		}

		_ = os.Setenv("SSH_CONNECTION", "10.0.0.1 1234 10.0.0.2 22")
		if IsDesktopGUI() {
			t.Errorf("expected IsDesktopGUI() to be false when SSH_CONNECTION is set")
		}

	default:
		// On windows and other platforms, IsDesktopGUI should execute without crashing
		_ = IsDesktopGUI()
	}
}

func TestOpenBrowserFunction(t *testing.T) {
	if runtime.GOOS == "linux" {
		origCmd := openBrowserCmd
		defer func() { openBrowserCmd = origCmd }()

		openBrowserCmd = func(url string) *exec.Cmd {
			return exec.Command("true")
		}

		if err := OpenBrowser("http://127.0.0.1:8080"); err != nil {
			t.Errorf("expected OpenBrowser to succeed with mock cmd, got: %v", err)
		}

		// Test default openBrowserCmd
		cmd := origCmd("http://127.0.0.1:8080")
		if cmd == nil || len(cmd.Args) < 2 || cmd.Args[0] != "xdg-open" {
			t.Errorf("unexpected cmd: %+v", cmd)
		}
	}
}


