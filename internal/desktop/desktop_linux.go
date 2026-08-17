//go:build linux

package desktop

import (
	"os"
	"os/exec"
)

var openBrowserCmd = func(url string) *exec.Cmd {
	return exec.Command("xdg-open", url)
}

func isDesktopGUIPlatform() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func openBrowserPlatform(url string) error {
	cmd := openBrowserCmd(url)
	return cmd.Start()
}

