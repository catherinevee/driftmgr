//go:build !windows
// +build !windows

package progress

import (
	"os"
	"strconv"
)

// getTerminalWidthPlatform returns the terminal width on Unix-like systems
func getTerminalWidthPlatform() int {
	width := 80
	if term := os.Getenv("TERM"); term != "" {
		// Try to get terminal width from environment
		if cols := os.Getenv("COLUMNS"); cols != "" {
			if w, err := strconv.Atoi(cols); err == nil {
				width = w
			}
		}
	}
	return width
}