//go:build !windows
// +build !windows

package bbs

import (
	"fmt"
	"net"
)

// CreateSocketFromHandle is a stub implementation for non-Windows platforms
// BBS doors with socket inheritance are primarily a Windows BBS feature
func CreateSocketFromHandle(socketHandle int) (net.Conn, error) {
	return nil, fmt.Errorf("socket handle inheritance not supported on this platform - this is a Windows BBS door feature")
}
