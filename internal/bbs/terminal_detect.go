package bbs

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

// DetectTerminalSize queries the terminal for its actual size using ANSI CPR
// This is the modern BBS approach - more reliable than dropfile values
// Returns (width, height, error)
func DetectTerminalSize(writer io.Writer, reader io.Reader) (int, int, error) {
	// Send Cursor Position Report (CPR) query: ESC[6n
	// Terminal responds with: ESC[{row};{col}R
	logrus.Debug("Querying terminal for size using ANSI CPR")

	// Send query
	_, err := writer.Write([]byte("\033[6n"))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to send CPR query: %w", err)
	}

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

	logrus.WithFields(logrus.Fields{
		"width":  cols,
		"height": rows,
	}).Info("Detected terminal size via ANSI CPR")

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
		// STDIO needs raw mode for ANSI query to work
		// Check if stdin is a terminal
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return 0, 0, fmt.Errorf("stdin is not a terminal (BBS pipe/socket mode)")
		}

		// Save current terminal state
		oldState, err := term.GetState(int(os.Stdin.Fd()))
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get terminal state: %w", err)
		}

		// Make terminal raw for the query
		_, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return 0, 0, fmt.Errorf("failed to make terminal raw: %w", err)
		}

		// Ensure we restore terminal state
		defer func() {
			if restoreErr := term.Restore(int(os.Stdin.Fd()), oldState); restoreErr != nil {
				logrus.WithError(restoreErr).Warn("Failed to restore terminal state")
			}
		}()

		// Use os.Stdin/Stdout directly for raw reads
		writer = os.Stdout
		reader = os.Stdin

		return DetectTerminalSize(writer, reader)

	default:
		return 0, 0, fmt.Errorf("unknown connection type")
	}
}
