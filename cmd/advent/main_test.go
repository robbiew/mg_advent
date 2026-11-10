package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robbiew/advent/internal/display"
	gd "github.com/robbiew/godoors"
)

func TestDoorProcess(t *testing.T) {
	if os.Getenv("MG_ADVENT_TEST_PROCESS") != "1" {
		t.Skip("subprocess helper")
	}
	main()
	os.Exit(0)
}

func runDoorProcess(t *testing.T, stdin string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	commandArgs := append([]string{"-test.run=^TestDoorProcess$"}, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], commandArgs...)
	cmd.Env = append(os.Environ(), "MG_ADVENT_TEST_PROCESS=1")
	cmd.Stdin = strings.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("door process: %v (%v), stderr: %s", err, ctx.Err(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected errors: %s", stderr.String())
	}
	return out
}

func dropFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "DOOR32.SYS")
	if err := os.WriteFile(path, []byte("0\r\n0\r\n57600\r\nTest BBS\r\n1\r\nReal Name\r\nCaller\r\n255\r\n42\r\n1\r\n3\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDoorNoDetectAndDisconnect(t *testing.T) {
	for _, keys := range []string{"q", ""} {
		out := runDoorProcess(t, keys, "-path", dropFile(t), "-nodetect", "-noice", "-debug-disable-date")
		if bytes.Contains(out, []byte("\x1b[6n")) || bytes.Contains(out, []byte("\x1b[c")) {
			t.Fatal("-nodetect sent a probe")
		}
		if bytes.Contains(out, []byte(display.DisableIceMode)) || bytes.Contains(out, []byte(display.EnableIceMode)) {
			t.Fatal("-noice changed ICE mode")
		}
		if !bytes.Contains(out, []byte(display.ShowCursor)) {
			t.Fatal("cursor was not restored")
		}
	}
}

func TestDoorProbeKeepsTypedQuit(t *testing.T) {
	out := runDoorProcess(t, "q\x1b[25;80R\x1b[?1;2c", "-path", dropFile(t), "-debug-disable-date")
	if !bytes.Contains(out, []byte("\x1b[6n")) {
		t.Fatal("expected size probe")
	}
	if !bytes.Contains(out, []byte(display.ShowCursor)) {
		t.Fatal("session did not finish")
	}
	withoutQuit := runDoorProcess(t, "\x1b[25;80R\x1b[?1;2c", "-path", dropFile(t), "-debug-disable-date")
	if bytes.Count(out, []byte(display.EraseScreen)) <= bytes.Count(withoutQuit, []byte(display.EraseScreen)) {
		t.Fatal("typed quit was lost during probe; goodbye screen was not shown")
	}

}

func TestDoorNavigationAndLogon(t *testing.T) {
	baseline := runDoorProcess(t, "q", "-local", "-nodetect", "-debug-disable-date")
	navigated := runDoorProcess(t, "\r\x1b[6~\x1b[5~\x1b[H\x1b[Fqq", "-local", "-nodetect", "-debug-disable-date")
	if bytes.Count(navigated, []byte(display.EraseScreen)) <= bytes.Count(baseline, []byte(display.EraseScreen)) {
		t.Fatal("navigation keys did not render new screens")
	}
	runDoorProcess(t, "q", "-path", dropFile(t), "-logon", "-nodetect", "-debug-disable-date")
}

func TestGoDoorsUserMetadata(t *testing.T) {
	info := gd.DropFile{Alias: "Caller", TimeLeft: 42, Emulation: 1, Node: 3}
	user := getUserInfo(false, &info)
	if user.Alias != "Caller" || user.TimeLeft != 42*time.Minute || user.NodeNum != 3 || user.Emulation != 1 {
		t.Fatalf("user: %+v", user)
	}
	if user := getUserInfo(true, &info); user.Alias != "SysOp" {
		t.Fatalf("local user: %+v", user)
	}
}

// artLine returns a middle line of an art file as the local-mode door renders it.
func artLine(t *testing.T, path string) []byte {
	t.Helper()
	de := newDisplayEngine(80, 25)
	lines, err := de.LoadAnsiLines(path)
	if err != nil || len(lines) < 6 {
		t.Fatalf("load %s: %v", path, err)
	}
	return []byte(lines[5])
}

func TestOffSeasonOpensArchive(t *testing.T) {
	*localMode = true
	defer func() { *localMode = false }()
	welcome := artLine(t, "art/2025/WELCOME.ANS")
	notYet := artLine(t, "art/common/NOTYET.ANS")

	out := runDoorProcess(t, "q", "-local", "-nodetect", "-debug-date=2026-07-04")
	if bytes.Contains(out, notYet) {
		t.Fatal("off-season door showed NOTYET instead of the archive")
	}
	if !bytes.Contains(out, welcome) {
		t.Fatal("off-season door did not open the newest past season (2025)")
	}
}

func TestLogonOffSeasonIsSilent(t *testing.T) {
	out := runDoorProcess(t, "", "-path", dropFile(t), "-logon", "-debug-date=2026-07-04")
	if len(out) != 0 {
		t.Fatalf("off-season logon wrote %d bytes; want nothing (no probe, no art)", len(out))
	}
}

// TestLogonIgnoresTypeAhead checks that a key already waiting when the door
// starts (the LF after the caller's Enter at the BBS) doesn't dismiss today's
// door, and that a real key afterwards does.
func TestLogonIgnoresTypeAhead(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestDoorProcess$",
		"-path", dropFile(t), "-logon", "-nodetect", "-debug-date=2025-12-03")
	cmd.Env = append(os.Environ(), "MG_ADVENT_TEST_PROCESS=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	stdin.Write([]byte("\n"))
	select {
	case err := <-done:
		t.Fatalf("logon exited on type-ahead (err %v); today's door was not shown", err)
	case <-time.After(1500 * time.Millisecond):
	}
	stdin.Write([]byte("x"))
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("door process: %v", err)
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("logon did not exit after a key press")
	}
	if !bytes.Contains(stdout.Bytes(), []byte(display.ShowCursor)) {
		t.Fatal("terminal was not restored")
	}
}

func TestPlausibleTermSize(t *testing.T) {
	for _, tc := range []struct {
		w, h int
		ok   bool
	}{
		{80, 25, true}, {132, 50, true}, {80, 24, true},
		{80, 12, false},   // half-height answer seen from mTelnet (issue #2)
		{40, 25, false},   // too narrow for the art
		{999, 999, false}, // terminal didn't clamp the probe cursor
	} {
		if got := plausibleTermSize(tc.w, tc.h); got != tc.ok {
			t.Errorf("plausibleTermSize(%d, %d) = %v, want %v", tc.w, tc.h, got, tc.ok)
		}
	}
}

func TestSessionLimitUsesBBSTimeLeft(t *testing.T) {
	if got := sessionLimit(display.User{TimeLeft: 42 * time.Minute}); got != 42*time.Minute {
		t.Fatalf("limit = %v, want the caller's 42 minutes", got)
	}
	if got := sessionLimit(display.User{TimeLeft: 600 * time.Minute}); got != maxSessionTime {
		t.Fatalf("limit = %v, want cap %v", got, maxSessionTime)
	}
	if got := sessionLimit(display.User{}); got != maxSessionTime {
		t.Fatalf("unknown time left: %v", got)
	}
}
