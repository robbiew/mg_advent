//go:build windows
// +build windows

package bbs

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procFreeConsole      = kernel32.NewProc("FreeConsole")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procGetStdHandle     = kernel32.NewProc("GetStdHandle")
	procGetFileType      = kernel32.NewProc("GetFileType")
	procSetConsoleMode   = kernel32.NewProc("SetConsoleMode")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
)

const (
	STD_INPUT_HANDLE                   = ^uint32(10) + 1
	STD_OUTPUT_HANDLE                  = ^uint32(11) + 1
	FILE_TYPE_PIPE                     = 3
	FILE_TYPE_CHAR                     = 2
	ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004 // Enable ANSI escape sequences
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

// EnableANSIProcessing enables ANSI escape sequence processing on Windows console
// This is required for Windows 7/8 to display ANSI art correctly
func EnableANSIProcessing() error {
	// Get stdout handle
	stdout, _, _ := procGetStdHandle.Call(uintptr(STD_OUTPUT_HANDLE))
	if stdout == 0 || stdout == uintptr(syscall.InvalidHandle) {
		return fmt.Errorf("failed to get stdout handle")
	}

	// Get current console mode
	var mode uint32
	ret, _, errno := procGetConsoleMode.Call(stdout, uintptr(unsafe.Pointer(&mode)))
	if ret == 0 {
		// Console mode not supported (might be redirected) - not an error
		return nil
	}

	// Enable virtual terminal processing (ANSI escape sequences)
	mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING
	ret, _, errno = procSetConsoleMode.Call(stdout, uintptr(mode))
	if ret == 0 {
		return fmt.Errorf("failed to enable ANSI processing: %v", errno)
	}

	return nil
}
