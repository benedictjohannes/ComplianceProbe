//go:build linux

package desktop

import (
	"os"
	"os/exec"

	"github.com/godbus/dbus/v5"
)

var openViaDBusPortal = func(url string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	options := map[string]dbus.Variant{
		"writable": dbus.MakeVariant(false),
		"ask":      dbus.MakeVariant(true),
	}
	token := os.Getenv("XDG_ACTIVATION_TOKEN")
	if token == "" {
		token = os.Getenv("DESKTOP_STARTUP_ID")
	}
	if token != "" {
		options["activation_token"] = dbus.MakeVariant(token)
	}

	call := obj.Call("org.freedesktop.portal.OpenURI.OpenURI", 0, "", url, options)
	return call.Err
}

var openBrowserCmd = func(url string) *exec.Cmd {
	return exec.Command("xdg-open", url)
}

func isDesktopGUIPlatform() bool {
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	// Also check if D-Bus session bus is available
	return os.Getenv("DBUS_SESSION_BUS_ADDRESS") != ""
}

func openBrowserPlatform(url string) error {
	// First, try opening via XDG Desktop Portal over D-Bus
	if err := openViaDBusPortal(url); err == nil {
		return nil
	}

	// Fallback to xdg-open
	cmd := openBrowserCmd(url)
	return cmd.Start()
}
