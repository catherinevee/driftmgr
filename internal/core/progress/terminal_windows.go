//go:build windows
// +build windows

package progress

import (
	"syscall"
	"unsafe"
)

// getTerminalWidthWindows returns the terminal width on Windows
func getTerminalWidthPlatform() int {
	type coord struct {
		x int16
		y int16
	}

	type small_rect struct {
		left   int16
		top    int16
		right  int16
		bottom int16
	}

	type consoleScreenBufferInfo struct {
		size              coord
		cursorPosition    coord
		attributes        uint16
		window            small_rect
		maximumWindowSize coord
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleScreenBufferInfo := kernel32.NewProc("GetConsoleScreenBufferInfo")

	handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return 80
	}

	var info consoleScreenBufferInfo
	ret, _, _ := getConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 80
	}

	return int(info.size.x)
}