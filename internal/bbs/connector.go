package bbs

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ConnectionType represents the type of BBS connection
type ConnectionType int

const (
	ConnectionStdio  ConnectionType = iota // Linux/Unix style STDIN/STDOUT
	ConnectionSocket                       // Windows style TCP socket
)

// BBSConnection handles BBS I/O operations
type BBSConnection struct {
	connType     ConnectionType
	socketConn   net.Conn
	stdinReader  *bufio.Reader
	stdoutWriter *bufio.Writer
	isConnected  bool
}

// NewBBSConnection creates a new BBS connection based on platform and dropfile
func NewBBSConnection(dropfilePath string) (*BBSConnection, error) {
	conn := &BBSConnection{}

	// Detect connection type based on platform
	if runtime.GOOS == "windows" {
		conn.connType = ConnectionSocket
		// Parse dropfile for socket information
		socketInfo, err := parseWindowsDropfile(dropfilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Windows dropfile: %w", err)
		}

		// Connect to socket
		socketConn, err := net.DialTimeout("tcp", socketInfo.Address, 10*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to BBS socket: %w", err)
		}

		conn.socketConn = socketConn
		conn.isConnected = true
		logrus.WithField("address", socketInfo.Address).Info("Connected to Windows BBS socket")

	} else {
		// Unix/Linux: Use STDIN/STDOUT
		conn.connType = ConnectionStdio
		conn.stdinReader = bufio.NewReader(os.Stdin)
		conn.stdoutWriter = bufio.NewWriter(os.Stdout)
		conn.isConnected = true
		logrus.Info("Using STDIN/STDOUT for BBS connection")
	}

	return conn, nil
}

// SocketInfo holds Windows socket connection information
type SocketInfo struct {
	Host    string
	Port    int
	Address string
}

// parseWindowsDropfile parses Windows-style dropfile for socket information
func parseWindowsDropfile(dropfilePath string) (*SocketInfo, error) {
	file, err := os.Open(dropfilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	info := &SocketInfo{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "SocketHost=") {
			info.Host = strings.TrimPrefix(line, "SocketHost=")
		} else if strings.HasPrefix(line, "SocketPort=") {
			portStr := strings.TrimPrefix(line, "SocketPort=")
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return nil, fmt.Errorf("invalid socket port: %s", portStr)
			}
			info.Port = port
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if info.Host == "" || info.Port == 0 {
		return nil, fmt.Errorf("socket information not found in dropfile")
	}

	info.Address = fmt.Sprintf("%s:%d", info.Host, info.Port)
	return info, nil
}

// Read reads data from the BBS connection
func (bc *BBSConnection) Read(p []byte) (n int, err error) {
	if !bc.isConnected {
		return 0, fmt.Errorf("not connected")
	}

	switch bc.connType {
	case ConnectionSocket:
		return bc.socketConn.Read(p)
	case ConnectionStdio:
		// For STDIN, we need to handle character-by-character reading
		// This is a simplified implementation
		if len(p) == 0 {
			return 0, nil
		}
		char, err := bc.stdinReader.ReadByte()
		if err != nil {
			return 0, err
		}
		p[0] = char
		return 1, nil
	default:
		return 0, fmt.Errorf("unsupported connection type")
	}
}

// Write writes data to the BBS connection
func (bc *BBSConnection) Write(p []byte) (n int, err error) {
	if !bc.isConnected {
		return 0, fmt.Errorf("not connected")
	}

	switch bc.connType {
	case ConnectionSocket:
		return bc.socketConn.Write(p)
	case ConnectionStdio:
		return bc.stdoutWriter.Write(p)
	default:
		return 0, fmt.Errorf("unsupported connection type")
	}
}

// Flush flushes any buffered output
func (bc *BBSConnection) Flush() error {
	if !bc.isConnected {
		return fmt.Errorf("not connected")
	}

	switch bc.connType {
	case ConnectionSocket:
		// Sockets are unbuffered, nothing to flush
		return nil
	case ConnectionStdio:
		return bc.stdoutWriter.Flush()
	default:
		return fmt.Errorf("unsupported connection type")
	}
}

// Close closes the BBS connection
func (bc *BBSConnection) Close() error {
	if !bc.isConnected {
		return nil
	}

	bc.isConnected = false

	switch bc.connType {
	case ConnectionSocket:
		if bc.socketConn != nil {
			return bc.socketConn.Close()
		}
	case ConnectionStdio:
		// STDIN/STDOUT don't need explicit closing
		if bc.stdoutWriter != nil {
			bc.stdoutWriter.Flush()
		}
	}

	return nil
}

// IsConnected returns whether the connection is active
func (bc *BBSConnection) IsConnected() bool {
	return bc.isConnected
}

// GetConnectionType returns the connection type
func (bc *BBSConnection) GetConnectionType() ConnectionType {
	return bc.connType
}

// ReadString reads a string from the BBS connection
func (bc *BBSConnection) ReadString() (string, error) {
	if !bc.isConnected {
		return "", fmt.Errorf("not connected")
	}

	switch bc.connType {
	case ConnectionSocket:
		reader := bufio.NewReader(bc.socketConn)
		return reader.ReadString('\n')
	case ConnectionStdio:
		return bc.stdinReader.ReadString('\n')
	default:
		return "", fmt.Errorf("unsupported connection type")
	}
}

// ReadByte reads a single byte from the BBS connection
func (bc *BBSConnection) ReadByte() (byte, error) {
	if !bc.isConnected {
		return 0, fmt.Errorf("not connected")
	}

	switch bc.connType {
	case ConnectionSocket:
		var b [1]byte
		_, err := bc.socketConn.Read(b[:])
		return b[0], err
	case ConnectionStdio:
		return bc.stdinReader.ReadByte()
	default:
		return 0, fmt.Errorf("unsupported connection type")
	}
}

// WriteString writes a string to the BBS connection
func (bc *BBSConnection) WriteString(s string) error {
	_, err := bc.Write([]byte(s))
	return err
}

// WriteByte writes a single byte to the BBS connection
func (bc *BBSConnection) WriteByte(b byte) error {
	_, err := bc.Write([]byte{b})
	return err
}
