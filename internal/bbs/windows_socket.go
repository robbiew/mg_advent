//go:build windows
// +build windows

package bbs

import (
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

var (
	ws2_32          = syscall.NewLazyDLL("ws2_32.dll")
	procSocket      = ws2_32.NewProc("socket")
	procBind        = ws2_32.NewProc("bind")
	procListen      = ws2_32.NewProc("listen")
	procAccept      = ws2_32.NewProc("accept")
	procClosesocket = ws2_32.NewProc("closesocket")
	procRecv        = ws2_32.NewProc("recv")
	procSend        = ws2_32.NewProc("send")
	procGetSockName = ws2_32.NewProc("getsockname")
	procGetPeerName = ws2_32.NewProc("getpeername")
	procIoctlsocket = ws2_32.NewProc("ioctlsocket")
)

const (
	AF_INET         = 2
	SOCK_STREAM     = 1
	IPPROTO_TCP     = 6
	SOL_SOCKET      = 0xffff
	SO_REUSEADDR    = 4
	FIONBIO         = 0x8004667e // Set non-blocking I/O mode
	WSAEWOULDBLOCK  = 10035      // Non-blocking socket operation would block
	WSAETIMEDOUT    = 10060      // Connection timed out
	WSAECONNRESET   = 10054      // Connection reset by peer
	WSAECONNABORTED = 10053      // Connection aborted
)

// WindowsSocket wraps a Windows socket handle for BBS door communication
type WindowsSocket struct {
	handle syscall.Handle
	fd     int
}

// CreateSocketFromHandle creates a net.Conn from an inherited Windows socket handle
func CreateSocketFromHandle(socketHandle int) (net.Conn, error) {
	logrus.WithField("handle", socketHandle).Info("Creating socket connection from inherited handle")

	// Convert the socket handle to a proper Windows handle
	handle := syscall.Handle(socketHandle)

	// CRITICAL: Set socket to BLOCKING mode (like ODoors does)
	// The inherited socket from BBS is likely non-blocking, causing WSAEWOULDBLOCK errors
	nonBlocking := uint32(0) // 0 = blocking, 1 = non-blocking
	ret, _, errno := procIoctlsocket.Call(
		uintptr(handle),
		uintptr(FIONBIO),
		uintptr(unsafe.Pointer(&nonBlocking)),
	)

	if ret == 0 {
		logrus.Info("Socket set to BLOCKING mode successfully")
	} else {
		logrus.WithField("errno", errno).Warn("Failed to set socket to blocking mode - proceeding anyway")
	}

	// Try to validate if it's a socket, but don't fail if validation fails
	// Many BBS systems provide redirected handles that aren't traditional sockets
	var sockaddr syscall.RawSockaddrAny
	sockaddrlen := int32(unsafe.Sizeof(sockaddr))

	ret, _, errno = procGetSockName.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&sockaddr)),
		uintptr(unsafe.Pointer(&sockaddrlen)),
	)

	if ret == 0 {
		logrus.WithField("handle", socketHandle).Info("Handle validated as socket")
	} else {
		logrus.WithFields(logrus.Fields{
			"handle": socketHandle,
			"errno":  errno,
		}).Warn("Handle validation failed - proceeding anyway (may be redirected handle)")
	}

	// Create a file descriptor from the Windows socket handle
	fd := int(handle)

	// Create a Go net.Conn wrapper around the Windows socket
	conn, err := createGoConnFromHandle(handle, fd)
	if err != nil {
		return nil, fmt.Errorf("failed to create Go connection from handle %d: %v", socketHandle, err)
	}

	logrus.WithField("handle", socketHandle).Info("Successfully created Go connection from Windows socket handle")
	return conn, nil
}

// createGoConnFromHandle creates a Go net.Conn from a Windows socket handle
func createGoConnFromHandle(handle syscall.Handle, fd int) (net.Conn, error) {
	// This is the most complex part - we need to create a net.Conn that works with Go's networking
	// We'll create a custom implementation that wraps the Windows socket operations

	socket := &WindowsSocket{
		handle: handle,
		fd:     fd,
	}

	return &WindowsSocketConn{socket: socket}, nil
}

// WindowsSocketConn implements net.Conn interface for Windows socket handles
type WindowsSocketConn struct {
	socket       *WindowsSocket
	localAddr    net.Addr
	remoteAddr   net.Addr
	readDeadline time.Time
}

// Read implements net.Conn.Read - socket is now in BLOCKING mode
func (c *WindowsSocketConn) Read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	// Socket is in blocking mode - this will wait until data arrives
	ret, _, errno := procRecv.Call(
		uintptr(c.socket.handle),
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)),
		0, // flags
	)

	if ret == ^uintptr(0) { // SOCKET_ERROR
		// Convert errno to syscall.Errno for comparison
		errNum := errno.(syscall.Errno)

		// Check for connection errors
		if errNum == WSAECONNRESET || errNum == WSAECONNABORTED || errNum == WSAETIMEDOUT {
			logrus.WithField("errno", errno).Info("Socket connection terminated")
			return 0, fmt.Errorf("socket recv failed: connection closed (%v)", errno)
		}

		// Other socket errors (shouldn't get WSAEWOULDBLOCK in blocking mode)
		return 0, fmt.Errorf("socket recv failed: %v", errno)
	}

	// Success - data received
	return int(ret), nil
}

// Write implements net.Conn.Write - socket is now in BLOCKING mode
func (c *WindowsSocketConn) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	// Socket is in blocking mode - this will wait until data can be sent
	ret, _, errno := procSend.Call(
		uintptr(c.socket.handle),
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)),
		0, // flags
	)

	if ret == ^uintptr(0) { // SOCKET_ERROR
		// Convert errno to syscall.Errno for comparison
		errNum := errno.(syscall.Errno)

		// Check for connection errors
		if errNum == WSAECONNRESET || errNum == WSAECONNABORTED || errNum == WSAETIMEDOUT {
			logrus.WithField("errno", errno).Info("Socket connection terminated")
			return 0, fmt.Errorf("socket send failed: connection closed (%v)", errno)
		}

		// Other socket errors (shouldn't get WSAEWOULDBLOCK in blocking mode)
		return 0, fmt.Errorf("socket send failed: %v", errno)
	}

	// Success - data sent
	return int(ret), nil
}

// Close implements net.Conn.Close
func (c *WindowsSocketConn) Close() error {
	ret, _, errno := procClosesocket.Call(uintptr(c.socket.handle))
	if ret != 0 {
		return fmt.Errorf("closesocket failed: %v", errno)
	}
	return nil
}

// LocalAddr implements net.Conn.LocalAddr
func (c *WindowsSocketConn) LocalAddr() net.Addr {
	if c.localAddr == nil {
		c.localAddr = c.getSocketAddr(true)
	}
	return c.localAddr
}

// RemoteAddr implements net.Conn.RemoteAddr
func (c *WindowsSocketConn) RemoteAddr() net.Addr {
	if c.remoteAddr == nil {
		c.remoteAddr = c.getSocketAddr(false)
	}
	return c.remoteAddr
}

// getSocketAddr gets the local or remote address of the socket
func (c *WindowsSocketConn) getSocketAddr(local bool) net.Addr {
	var sockaddr syscall.RawSockaddrAny
	sockaddrlen := int32(unsafe.Sizeof(sockaddr))

	if local {
		ret, _, err := procGetSockName.Call(
			uintptr(c.socket.handle),
			uintptr(unsafe.Pointer(&sockaddr)),
			uintptr(unsafe.Pointer(&sockaddrlen)),
		)
		if ret != 0 {
			logrus.WithError(err).Warn("Failed to get local socket address")
			return &net.TCPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0}
		}
	} else {
		ret, _, err := procGetPeerName.Call(
			uintptr(c.socket.handle),
			uintptr(unsafe.Pointer(&sockaddr)),
			uintptr(unsafe.Pointer(&sockaddrlen)),
		)
		if ret != 0 {
			logrus.WithError(err).Warn("Failed to get remote socket address")
			return &net.TCPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0}
		}
	}

	// Parse the sockaddr structure
	if sockaddr.Addr.Family == AF_INET {
		// IPv4 address
		addr := (*syscall.RawSockaddrInet4)(unsafe.Pointer(&sockaddr))
		ip := net.IPv4(addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3])
		port := int(addr.Port>>8) | int(addr.Port&0xff)<<8 // Convert from network byte order
		return &net.TCPAddr{IP: ip, Port: port}
	}

	// Fallback
	return &net.TCPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0}
}

// SetDeadline implements net.Conn.SetDeadline
func (c *WindowsSocketConn) SetDeadline(t time.Time) error {
	// TODO: Implement deadline support if needed
	return nil
}

// SetReadDeadline implements net.Conn.SetReadDeadline
func (c *WindowsSocketConn) SetReadDeadline(t time.Time) error {
	// TODO: Implement read deadline support if needed
	return nil
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline
func (c *WindowsSocketConn) SetWriteDeadline(t time.Time) error {
	// TODO: Implement write deadline support if needed
	return nil
}
