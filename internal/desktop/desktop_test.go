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
		// Test when DISPLAY / WAYLAND_DISPLAY / DBUS_SESSION_BUS_ADDRESS are unset
		origDisplay := os.Getenv("DISPLAY")
		origWayland := os.Getenv("WAYLAND_DISPLAY")
		origDBus := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
		defer func() {
			_ = os.Setenv("DISPLAY", origDisplay)
			_ = os.Setenv("WAYLAND_DISPLAY", origWayland)
			_ = os.Setenv("DBUS_SESSION_BUS_ADDRESS", origDBus)
		}()

		_ = os.Unsetenv("DISPLAY")
		_ = os.Unsetenv("WAYLAND_DISPLAY")
		_ = os.Unsetenv("DBUS_SESSION_BUS_ADDRESS")
		if IsDesktopGUI() {
			t.Errorf("expected IsDesktopGUI() to be false when DISPLAY, WAYLAND_DISPLAY, and DBUS_SESSION_BUS_ADDRESS are unset")
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

		_ = os.Unsetenv("WAYLAND_DISPLAY")
		_ = os.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/1000/bus")
		if !IsDesktopGUI() {
			t.Errorf("expected IsDesktopGUI() to be true when DBUS_SESSION_BUS_ADDRESS is set")
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
		origPortal := openViaDBusPortal
		defer func() {
			openBrowserCmd = origCmd
			openViaDBusPortal = origPortal
		}()

		// Test successful D-Bus portal call
		portalCalled := false
		openViaDBusPortal = func(url string) error {
			portalCalled = true
			return nil
		}

		if err := openBrowserPlatform("http://127.0.0.1:8080"); err != nil {
			t.Errorf("expected openBrowserPlatform to succeed with portal, got: %v", err)
		}
		if !portalCalled {
			t.Errorf("expected openViaDBusPortal to be called")
		}

		// Test fallback to xdg-open when D-Bus fails
		openViaDBusPortal = func(url string) error {
			return os.ErrNotExist
		}
		cmdCalled := false
		openBrowserCmd = func(url string) *exec.Cmd {
			cmdCalled = true
			return exec.Command("true")
		}

		if err := openBrowserPlatform("http://127.0.0.1:8080"); err != nil {
			t.Errorf("expected openBrowserPlatform to succeed with fallback, got: %v", err)
		}
		if !cmdCalled {
			t.Errorf("expected openBrowserCmd fallback to be called")
		}
	}
}


