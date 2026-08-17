//go:build !linux && !darwin && !windows

package desktop

import "fmt"

func isDesktopGUIPlatform() bool {
	return false
}

func openBrowserPlatform(url string) error {
	return fmt.Errorf("browser launch not supported on this platform")
}
