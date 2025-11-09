//go:build windows
// +build windows

package bbs

import (
	"syscall"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procFreeConsole      = kernel32.NewProc("FreeConsole")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procGetStdHandle     = kernel32.NewProc("GetStdHandle")
	procGetFileType      = kernel32.NewProc("GetFileType")
)

const (
	STD_INPUT_HANDLE  = ^uint32(10) + 1
	STD_OUTPUT_HANDLE = ^uint32(11) + 1
	FILE_TYPE_PIPE    = 3
	FILE_TYPE_CHAR    = 2
)

// DetectBBSMode determines if we're running under BBS redirection
func DetectBBSMode() bool {
	// Get standard handles
	stdIn, _, _ := procGetStdHandle.Call(uintptr(STD_INPUT_HANDLE))
	stdOut, _, _ := procGetStdHandle.Call(uintptr(STD_OUTPUT_HANDLE))

	// Check handle types
	inType, _, _ := procGetFileType.Call(stdIn)
	outType, _, _ := procGetFileType.Call(stdOut)

	// If both are pipes, we're likely running under BBS redirection
	return inType == FILE_TYPE_PIPE && outType == FILE_TYPE_PIPE
}

// FreeConsoleIfBBS frees the console if we detect BBS mode
func FreeConsoleIfBBS() bool {
	if DetectBBSMode() {
		// Free the console so we inherit BBS handles properly
		procFreeConsole.Call()
		return true
	}
	return false
}

// HasConsole checks if we have a console window
func HasConsole() bool {
	consoleWindow, _, _ := procGetConsoleWindow.Call()
	return consoleWindow != 0
}
