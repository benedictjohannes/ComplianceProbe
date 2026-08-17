package desktop

// IsDesktopGUI returns true if the current environment is detected to be
// an interactive graphical desktop session capable of displaying a web browser.
func IsDesktopGUI() bool {
	return isDesktopGUIPlatform()
}

// OpenBrowser attempts to open the specified URL in the default web browser.
// This is best-effort and non-fatal.
func OpenBrowser(url string) error {
	return openBrowserPlatform(url)
}
