package bbs

import (
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
)

// DetectTerminalSize queries the terminal for its actual size using ANSI escape sequences
// This method moves cursor to far bottom-right, queries position, then restores cursor
// Returns (width, height, error)
func DetectTerminalSize(writer io.Writer, reader io.Reader) (int, int, error) {
	logrus.Debug("Detecting terminal size using cursor positioning method")

	// Helper function to flush buffered writers
	flushWriter := func() error {
		if flusher, ok := writer.(interface{ Flush() error }); ok {
			return flusher.Flush()
		}
		return nil
	}

	// Step 0: Clear screen first for clean detection environment
	_, err := writer.Write([]byte("\033[2J\033[H")) // Clear screen and move to home
	if err != nil {
		return 0, 0, fmt.Errorf("failed to clear screen initially: %w", err)
	}
	flushWriter() // Ensure screen is cleared before detection

	// Step 1: Save current cursor position
	_, err = writer.Write([]byte("\033[s")) // Save cursor position
	if err != nil {
		return 0, 0, fmt.Errorf("failed to save cursor position: %w", err)
	}

	// Step 2: Move cursor to far bottom-right (terminal will clamp to actual size)
	_, err = writer.Write([]byte("\033[999;999H")) // Move to row 999, col 999
	if err != nil {
		return 0, 0, fmt.Errorf("failed to move cursor: %w", err)
	}

	// Step 3: Make any response invisible by setting text color to black
	_, err = writer.Write([]byte("\033[30m")) // Set foreground color to black (invisible)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to set invisible color: %w", err)
	}

	// Step 4: Query current cursor position (will be clamped to actual terminal size)
	_, err = writer.Write([]byte("\033[6n")) // Query cursor position
	if err != nil {
		return 0, 0, fmt.Errorf("failed to send CPR query: %w", err)
	}

	// CRITICAL: Flush buffered output before reading response
	// This is essential for Linux STDIO connections using bufio.Writer
	if err := flushWriter(); err != nil {
		return 0, 0, fmt.Errorf("failed to flush output: %w", err)
	}
	logrus.Debug("Flushed output buffer before reading CPR response")

	// Read response with timeout
	response := make([]byte, 32)
	done := make(chan error, 1)
	var n int

	go func() {
		var readErr error
		n, readErr = reader.Read(response)
		done <- readErr
	}()

	select {
	case err := <-done:
		if err != nil {
			return 0, 0, fmt.Errorf("failed to read CPR response: %w", err)
		}
	case <-time.After(1 * time.Second):
		return 0, 0, fmt.Errorf("timeout waiting for CPR response")
	}

	// Parse response: ESC[{row};{col}R
	// Example: \033[25;80R means 25 rows, 80 columns

	// Bounds check to prevent slice panic
	if n < 0 || n > len(response) {
		return 0, 0, fmt.Errorf("invalid response length: %d (buffer size: %d)", n, len(response))
	}

	responseStr := string(response[:n])
	logrus.WithField("response", responseStr).Debug("Received CPR response")

	re := regexp.MustCompile(`\033\[(\d+);(\d+)R`)
	matches := re.FindStringSubmatch(responseStr)

	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("invalid CPR response format: %s", responseStr)
	}

	var rows, cols int
	if _, err := fmt.Sscanf(matches[1], "%d", &rows); err != nil {
		return 0, 0, fmt.Errorf("failed to parse rows: %w", err)
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &cols); err != nil {
		return 0, 0, fmt.Errorf("failed to parse columns: %w", err)
	}

	// Step 5: Restore colors, clear screen, and restore cursor position
	writer.Write([]byte("\033[0m")) // Reset all attributes (color, bold, etc.) to default
	writer.Write([]byte("\033[2K")) // Clear the current line (where CPR response appeared)
	writer.Write([]byte("\033[u"))  // Restore original cursor position
	writer.Write([]byte("\033[2J")) // Clear entire screen immediately after detection
	writer.Write([]byte("\033[H"))  // Move cursor to home position (1,1)
	flushWriter()                   // Ensure all output is sent

	logrus.WithFields(logrus.Fields{
		"width":  cols,
		"height": rows,
	}).Info("Detected terminal size via cursor positioning method")

	return cols, rows, nil
}

// DetectTerminalSizeFromConnection wraps DetectTerminalSize for BBSConnection
// Handles raw terminal mode for STDIO connections
func (c *BBSConnection) DetectTerminalSize() (int, int, error) {
	if !c.isConnected {
		return 0, 0, fmt.Errorf("not connected")
	}

	var writer io.Writer
	var reader io.Reader

	switch c.connType {
	case ConnectionSocket:
		// Socket connections are already raw
		writer = c.socketConn
		reader = c.socketConn
		return DetectTerminalSize(writer, reader)

	case ConnectionStdio:
		// Linux BBS mode - uses STDIN/STDOUT pipes, no raw mode needed
		// The BBS handles the terminal and forwards ANSI queries to the user's terminal
		logrus.Debug("Linux BBS mode (STDIO pipes) - using buffered I/O for size detection")

		// Use the buffered readers/writers from BBSConnection
		// The BBS will forward ANSI escape sequences to the user's terminal and back
		writer = c.stdoutWriter
		reader = c.stdinReader

		return DetectTerminalSize(writer, reader)

	default:
		return 0, 0, fmt.Errorf("unknown connection type")
	}
}
