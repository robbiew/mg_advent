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

// NewBBSConnectionFromSocket creates a BBS connection from a socket handle passed on command line
// This is faster than reading from door32.sys and is Mystic's recommended method
func NewBBSConnectionFromSocket(socketHandle int, dropfilePath string) (*BBSConnection, error) {
	conn := &BBSConnection{}

	logrus.WithField("socketHandle", socketHandle).Info("Creating BBS connection directly from socket handle")

	// Create socket connection from handle
	socketConn, err := CreateSocketFromHandle(socketHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket from handle %d: %w", socketHandle, err)
	}

	logrus.Info("Socket connection created successfully")

	// No stabilization delay needed - socket is already ready
	conn.connType = ConnectionSocket
	conn.socketConn = socketConn
	conn.isConnected = true

	logrus.Info("BBS connection initialized from command-line socket handle")
	return conn, nil
}

// NewBBSConnection creates a new BBS connection based on platform and dropfile
func NewBBSConnection(dropfilePath string) (*BBSConnection, error) {
	conn := &BBSConnection{}

	// Detect connection type based on platform
	if runtime.GOOS == "windows" {
		// Parse dropfile to determine connection type
		door32Info, err := ParseDoor32(dropfilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse door32.sys: %w", err)
		}

		if door32Info.LineType == 2 {
			// Socket/telnet connection - use handle inheritance (proper implementation)
			conn.connType = ConnectionSocket

			logrus.WithFields(logrus.Fields{
				"socketHandle": door32Info.SocketHandle,
				"address":      door32Info.Socket.Address,
			}).Info("Attempting to inherit socket handle from parent BBS process")

			// Try to create connection from inherited socket handle (proper Door32 method)
			socketConn, err := CreateSocketFromHandle(door32Info.SocketHandle)
			if err != nil {
				logrus.WithError(err).Warn("Socket handle inheritance failed, trying TCP connection fallback")

				// Fallback: attempt new TCP connection
				if door32Info.Socket != nil {
					logrus.WithField("address", door32Info.Socket.Address).Info("Attempting TCP connection fallback to BBS")
					socketConn, err = net.DialTimeout("tcp", door32Info.Socket.Address, 10*time.Second)
					if err != nil {
						logrus.WithError(err).Error("Both socket handle inheritance and TCP connection failed")
						return nil, fmt.Errorf("failed to connect to BBS - handle inheritance failed: %v, TCP connection to %s also failed: %w",
							err, door32Info.Socket.Address, err)
					}
					logrus.WithField("address", door32Info.Socket.Address).Info("Successfully connected to BBS via TCP fallback")
				} else {
					return nil, fmt.Errorf("socket handle inheritance failed and no socket information available for TCP fallback: %v", err)
				}
			} else {
				logrus.WithField("handle", door32Info.SocketHandle).Info("Successfully inherited socket handle from parent BBS process")
			}

			// Give Windows 7 BBS systems time to fully establish the connection
			// This prevents socket WSAEWOULDBLOCK errors on first write
			logrus.Info("Starting 200ms socket stabilization delay")
			time.Sleep(200 * time.Millisecond)
			logrus.Info("Socket stabilization delay completed")

			conn.socketConn = socketConn
			conn.isConnected = true

			// CRITICAL: Send null byte to "wake up" Mystic BBS output buffering
			// Mystic on Windows 7 buffers door output until it receives data
			logrus.Info("Sending wake-up byte to Mystic BBS")
			socketConn.Write([]byte{0})
			time.Sleep(50 * time.Millisecond) // Give Mystic time to process
			logrus.Info("BBSConnection initialization complete")
		} else {
			// Local or serial mode
			conn.connType = ConnectionStdio
			conn.stdinReader = bufio.NewReader(os.Stdin)
			conn.stdoutWriter = bufio.NewWriter(os.Stdout)
			conn.isConnected = true
			logrus.Info("Using STDIN/STDOUT for local/serial connection")
		}

	} else {
		// Unix/Linux: Always use STDIN/STDOUT
		// The BBS system handles the socket and provides stdin/stdout to the door
		conn.connType = ConnectionStdio
		conn.stdinReader = bufio.NewReader(os.Stdin)
		conn.stdoutWriter = bufio.NewWriter(os.Stdout)
		conn.isConnected = true
		logrus.Info("Using STDIN/STDOUT for Linux BBS (door32.sys parsed for user info only)")
	}

	return conn, nil
}

// ParseDoor32 parses a door32.sys file and returns user information
// Follows the official Door32 specification (11 lines)
func ParseDoor32(dropfilePath string) (*Door32Info, error) {
	file, err := os.Open(dropfilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := []string{}

	// Read all lines
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	logrus.WithField("lines", len(lines)).Info("Parsing door32.sys file")
	for i, line := range lines {
		logrus.WithFields(logrus.Fields{
			"line":    i + 1,
			"content": line,
		}).Debug("door32.sys line")
	}

	if len(lines) < 11 {
		return nil, fmt.Errorf("door32.sys file is incomplete, expected 11 lines per spec, got %d", len(lines))
	}

	info := &Door32Info{}

	// Parse according to official Door32 specification:
	// Line 1 : Comm type (0=local, 1=serial, 2=telnet)
	if info.LineType, err = strconv.Atoi(lines[0]); err != nil {
		return nil, fmt.Errorf("invalid comm type (line 1): %s", lines[0])
	}

	// Line 2 : Comm or socket handle
	if info.SocketHandle, err = strconv.Atoi(lines[1]); err != nil {
		return nil, fmt.Errorf("invalid socket handle (line 2): %s", lines[1])
	}

	// Line 3 : Baud rate (we don't use this for socket connections)
	// Line 4 : BBSID (software name and version)
	info.BBSName = lines[3]

	// Line 5 : User record position (1-based) - we don't use this
	// Line 6 : User's real name
	realName := lines[5]
	nameParts := strings.Fields(realName)
	if len(nameParts) >= 2 {
		info.FirstName = nameParts[0]
		info.LastName = strings.Join(nameParts[1:], " ")
	} else {
		info.FirstName = realName
		info.LastName = ""
	}

	// Line 7 : User's handle/alias
	info.Alias = lines[6]

	// Line 8 : User's security level
	if info.SecurityLevel, err = strconv.Atoi(lines[7]); err != nil {
		return nil, fmt.Errorf("invalid security level (line 8): %s", lines[7])
	}

	// Line 9 : User's time left (in minutes)
	if info.TimeLeft, err = strconv.Atoi(lines[8]); err != nil {
		return nil, fmt.Errorf("invalid time left (line 9): %s", lines[8])
	}

	// Line 10: Emulation (0=Ascii, 1=Ansi, 2=Avatar, 3=RIP, 4=Max Graphics)
	if info.Emulation, err = strconv.Atoi(lines[9]); err != nil {
		return nil, fmt.Errorf("invalid emulation (line 10): %s", lines[9])
	}

	// Line 11: Current node number
	if info.NodeNumber, err = strconv.Atoi(lines[10]); err != nil {
		return nil, fmt.Errorf("invalid node number (line 11): %s", lines[10])
	}

	// Parse socket information if it's a telnet connection (Windows only)
	if info.LineType == 2 && runtime.GOOS == "windows" {
		socketInfo, err := parseWindowsDropfile(dropfilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse socket info: %w", err)
		}
		info.Socket = socketInfo
	}

	logrus.WithFields(logrus.Fields{
		"commType":     info.LineType,
		"socketHandle": info.SocketHandle,
		"bbsName":      info.BBSName,
		"realName":     info.FirstName + " " + info.LastName,
		"alias":        info.Alias,
		"security":     info.SecurityLevel,
		"timeLeft":     info.TimeLeft,
		"emulation":    info.Emulation,
		"node":         info.NodeNumber,
	}).Info("Parsed door32.sys according to official specification")

	return info, nil
}

// SocketInfo holds Windows socket connection information
type SocketInfo struct {
	Host    string
	Port    int
	Address string
}

// Door32Info holds parsed information from door32.sys
type Door32Info struct {
	LineType      int
	BBSName       string
	FirstName     string
	LastName      string
	Alias         string
	SecurityLevel int
	TimeLeft      int
	Emulation     int
	NodeNumber    int
	SocketHandle  int
	Socket        *SocketInfo
}

// parseWindowsDropfile parses door32.sys dropfile for socket information
func parseWindowsDropfile(dropfilePath string) (*SocketInfo, error) {
	file, err := os.Open(dropfilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	info := &SocketInfo{}
	lines := []string{}

	// Read all lines first
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) < 2 {
		return nil, fmt.Errorf("door32.sys file is too short for socket parsing")
	}

	// Parse line type
	lineType, err := strconv.Atoi(lines[0])
	if err != nil {
		return nil, fmt.Errorf("invalid line type in door32.sys: %s", lines[0])
	}

	if lineType != 2 {
		return nil, fmt.Errorf("unsupported door32.sys line type: %d (expected 2 for socket)", lineType)
	}

	// Parse socket handle from line 2 according to Door32 specification
	socketHandle, err := strconv.Atoi(lines[1])
	if err != nil {
		return nil, fmt.Errorf("invalid socket handle (line 2): %s", lines[1])
	}

	// Use default socket host (localhost)
	info.Host = "127.0.0.1"
	info.Port = socketHandle

	// Look for custom socket info at the end of the file (our batch file adds this)
	for i := 10; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "SocketHost=") {
			info.Host = strings.TrimPrefix(line, "SocketHost=")
		} else if strings.HasPrefix(line, "SocketPort=") {
			portStr := strings.TrimPrefix(line, "SocketPort=")
			port, err := strconv.Atoi(portStr)
			if err != nil {
				logrus.WithError(err).Warn("Invalid custom socket port, using handle as port")
			} else {
				info.Port = port
			}
		}
	}

	if info.Port == 0 {
		return nil, fmt.Errorf("no valid socket port found in door32.sys")
	}

	info.Address = fmt.Sprintf("%s:%d", info.Host, info.Port)
	logrus.WithFields(logrus.Fields{
		"host":    info.Host,
		"port":    info.Port,
		"address": info.Address,
	}).Info("Parsed door32.sys socket information")

	return info, nil
} // Read reads data from the BBS connection
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

// Flush flushes any buffered output AND waits for it to transmit
func (bc *BBSConnection) Flush() error {
	if !bc.isConnected {
		return fmt.Errorf("not connected")
	}

	switch bc.connType {
	case ConnectionSocket:
		// CRITICAL: Windows sockets need time to transmit buffered data
		// Wait up to 100ms for output to drain (like ODoors' ODWaitDrain)
		time.Sleep(100 * time.Millisecond)
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
