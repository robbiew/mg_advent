package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/robbiew/advent/internal/bbs"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_socket_inheritance <door32.sys_path>")
		os.Exit(1)
	}

	door32Path := os.Args[1]
	if !filepath.IsAbs(door32Path) {
		cwd, _ := os.Getwd()
		door32Path = filepath.Join(cwd, door32Path)
	}

	fmt.Printf("Testing socket handle inheritance with: %s\n", door32Path)

	// Parse the door32.sys file
	info, err := bbs.ParseDoor32(door32Path, "192.168.1.131")
	if err != nil {
		log.Fatalf("Failed to parse door32.sys: %v", err)
	}

	fmt.Printf("Door32 Info:\n")
	fmt.Printf("  Line Type: %d\n", info.LineType)
	fmt.Printf("  Socket Handle: %d\n", info.SocketHandle)
	fmt.Printf("  User: %s %s (%s)\n", info.FirstName, info.LastName, info.Alias)
	fmt.Printf("  Time Left: %d minutes\n", info.TimeLeft)
	fmt.Printf("  Node: %d\n", info.NodeNumber)

	if info.Socket != nil {
		fmt.Printf("  Socket: %s\n", info.Socket.Address)
	}

	// If it's a socket connection, test handle inheritance
	if info.LineType == 2 {
		fmt.Printf("\nTesting Windows socket handle inheritance...\n")

		conn, err := bbs.CreateSocketFromHandle(info.SocketHandle)
		if err != nil {
			fmt.Printf("Socket handle inheritance failed: %v\n", err)
			fmt.Printf("This is expected when not running from a BBS that provides a valid socket handle.\n")
		} else {
			fmt.Printf("Socket handle inheritance SUCCESS!\n")
			fmt.Printf("Local address: %s\n", conn.LocalAddr())
			fmt.Printf("Remote address: %s\n", conn.RemoteAddr())
			conn.Close()
		}
	} else {
		fmt.Printf("Not a socket connection (line type %d), skipping socket tests.\n", info.LineType)
	}

	fmt.Printf("\nTest completed.\n")
}
