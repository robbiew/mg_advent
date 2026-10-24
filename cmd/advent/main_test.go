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
