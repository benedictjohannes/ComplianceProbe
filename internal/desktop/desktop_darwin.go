//go:build darwin

package desktop

import (
	"os"
	"os/exec"
)

func isDesktopGUIPlatform() bool {
	// Best-effort desktop user session hint (not headless SSH)
	isSSH := os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != ""
	return !isSSH
}

func openBrowserPlatform(url string) error {
	cmd := exec.Command("/usr/bin/open", url)
	return cmd.Start()
}
