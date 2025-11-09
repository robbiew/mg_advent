package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	fmt.Println("=== BBS Door Console Diagnostics ===")

	// Check if we have a console allocated
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	procGetStdHandle := kernel32.NewProc("GetStdHandle")

	// Get console window handle
	consoleWindow, _, _ := procGetConsoleWindow.Call()
	fmt.Printf("Console Window Handle: 0x%x\n", consoleWindow)

	if consoleWindow == 0 {
		fmt.Println("No console window - running in background/redirected mode")
	} else {
		fmt.Println("Console window exists - may need to free it for BBS mode")
	}

	// Check standard handles
	stdInputHandle, _, _ := procGetStdHandle.Call(uintptr(^uint32(10) + 1))  // STD_INPUT_HANDLE
	stdOutputHandle, _, _ := procGetStdHandle.Call(uintptr(^uint32(11) + 1)) // STD_OUTPUT_HANDLE
	stdErrorHandle, _, _ := procGetStdHandle.Call(uintptr(^uint32(12) + 1))  // STD_ERROR_HANDLE

	fmt.Printf("Stdin Handle: 0x%x\n", stdInputHandle)
	fmt.Printf("Stdout Handle: 0x%x\n", stdOutputHandle)
	fmt.Printf("Stderr Handle: 0x%x\n", stdErrorHandle)

	// Check if stdin/stdout are pipes (BBS redirection)
	fileType1, _, _ := kernel32.NewProc("GetFileType").Call(stdInputHandle)
	fileType2, _, _ := kernel32.NewProc("GetFileType").Call(stdOutputHandle)

	fmt.Printf("Stdin Type: %d (1=disk, 2=char/console, 3=pipe)\n", fileType1)
	fmt.Printf("Stdout Type: %d (1=disk, 2=char/console, 3=pipe)\n", fileType2)

	if fileType1 == 3 && fileType2 == 3 {
		fmt.Println("✓ Both stdin/stdout are pipes - BBS redirection detected!")
		fmt.Println("Door should work in this mode")
	} else if fileType1 == 2 && fileType2 == 2 {
		fmt.Println("✓ Both stdin/stdout are console - local testing mode")
	} else {
		fmt.Println("? Mixed handle types - check BBS configuration")
	}

	// Test basic I/O
	fmt.Println("\n=== I/O Test ===")
	fmt.Print("This should appear in user's terminal if BBS redirection works.\n")
	fmt.Print("Press any key to continue...")

	var input [1]byte
	os.Stdin.Read(input[:])

	fmt.Printf("\nReceived: 0x%02x (%q)\n", input[0], input[0])
	fmt.Println("=== End Diagnostics ===")
}
