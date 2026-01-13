package main

import (
	"os"
	"strings"

	"github.com/acarl005/stripansi"
)

func cleanupOutput(input string) string {
	// 1. Use stripansi to handle cross-platform ANSI/CSI/OSC sequences robustly
	// This catches the 'rubbish' from Windows and macOS that regex might miss.
	output := stripansi.Strip(input)

	// 2. Explicitly remove BEL and other non-printable control chars
	// that aren't always part of escape sequences.
	output = strings.Map(func(r rune) rune {
		if r == '\u0007' || (r < 32 && r != '\n' && r != '\r' && r != '\t') {
			return -1 // Drop the character
		}
		return r
	}, output)

	return strings.TrimSpace(output)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
