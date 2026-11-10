package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robbiew/advent/internal/display"
	gd "github.com/robbiew/godoors"
	"golang.org/x/text/encoding/charmap"
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
	return dropFileWithTime(t, 42)
}

// dropFileWithTime writes a door32.sys giving the caller minutesLeft minutes.
func dropFileWithTime(t *testing.T, minutesLeft int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "DOOR32.SYS")
	body := fmt.Sprintf("0\r\n0\r\n57600\r\nTest BBS\r\n1\r\nReal Name\r\nCaller\r\n255\r\n%d\r\n1\r\n3\r\n", minutesLeft)
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
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
		if bytes.Contains(out, []byte(display.BlinkMode)) {
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

// artLine returns a middle line of an art file as the local-mode door renders
// it: the engine keeps CP437 and converts to UTF-8 on output.
func artLine(t *testing.T, path string) []byte {
	t.Helper()
	de := newDisplayEngine(80, 25)
	lines, err := de.LoadAnsiLines(path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	sample := ""
	if len(lines) >= 6 {
		sample = lines[5]
	} else {
		// Art drawn for autowrap has a few very long lines; take a stretch
		// from the middle of the longest (CP437 is one byte per character).
		longest := ""
		for _, l := range lines {
			if len(l) > len(longest) {
				longest = l
			}
		}
		if len(longest) < 200 {
			t.Fatalf("%s: too little art to sample", path)
		}
		mid := len(longest) / 2
		sample = longest[mid : mid+80]
	}
	line, err := charmap.CodePage437.NewDecoder().Bytes([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	return line
}

func TestOffSeasonOpensArchive(t *testing.T) {
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
	// An explicit zero from door32.sys means the caller is out of time.
	if got := sessionLimit(getUserInfo(false, &gd.DropFile{TimeLeft: 0})); got != 0 {
		t.Fatalf("zero minutes left: limit = %v, want 0", got)
	}
	if got := sessionLimit(display.User{TimeLeft: -5 * time.Minute}); got != 0 {
		t.Fatalf("negative time left: limit = %v, want 0", got)
	}
}

func TestDoorWithNoTimeLeftReturnsToBBS(t *testing.T) {
	out := runDoorProcess(t, "", "-path", dropFileWithTime(t, 0), "-nodetect", "-debug-disable-date")
	if !bytes.Contains(out, []byte("Your time is up")) {
		t.Fatal("caller with no time left was not told their time is up")
	}
	if bytes.Contains(out, []byte(display.HideCursor)) {
		t.Fatal("door started the session (drew art) for a caller with no time left")
	}
}

// A new season's art ships before December: it must stay hidden until Dec 1
// and open as the live season on the day.
func TestNewSeasonOpensOnDecemberFirst(t *testing.T) {
	welcome2026 := artLine(t, "art/2026/WELCOME.ANS")
	before := runDoorProcess(t, "q", "-local", "-nodetect", "-debug-date=2026-11-10")
	if bytes.Contains(before, welcome2026) {
		t.Fatal("2026 art was visible in November")
	}
	during := runDoorProcess(t, "q", "-local", "-nodetect", "-debug-date=2026-12-03")
	if !bytes.Contains(during, welcome2026) {
		t.Fatal("2026 did not open as the live season on Dec 3")
	}
}

// runDoorScript runs the door with keys sent over time: each step is a
// string to type or a time.Duration to wait. Stdin closes after the last step.
func runDoorScript(t *testing.T, args []string, steps ...any) []byte {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestDoorProcess$"}, args...)...)
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
	for _, step := range steps {
		switch s := step.(type) {
		case string:
			stdin.Write([]byte(s))
		case time.Duration:
			time.Sleep(s)
		}
	}
	stdin.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("door process: %v", err)
		}
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		t.Fatal("door did not exit")
	}
	return stdout.Bytes()
}

func TestHelpScreen(t *testing.T) {
	out := runDoorProcess(t, "?xq", "-local", "-nodetect", "-debug-date=2026-07-04")
	if !bytes.Contains(out, []byte("Advent Calendar keys")) || !bytes.Contains(out, []byte("type a door number")) {
		t.Fatal("? did not show the key help")
	}
}

func TestOffSeasonCountdown(t *testing.T) {
	out := runDoorProcess(t, "q", "-local", "-nodetect", "-debug-date=2026-11-10")
	if !bytes.Contains(out, []byte("The 2026 calendar opens in 21 days")) {
		t.Fatal("welcome screen did not show the countdown")
	}
	if out := runDoorProcess(t, "q", "-local", "-nodetect", "-debug-date=2026-12-03"); bytes.Contains(out, []byte("calendar opens")) {
		t.Fatal("countdown shown during the season")
	}
}

func TestArtistCredits(t *testing.T) {
	// 2025 door 1 is "Lamplight" by LDA / Mistigris.
	out := runDoorProcess(t, "\raxqq", "-local", "-nodetect", "-debug-date=2026-07-04")
	for _, want := range []string{"About this art", "Lamplight", "LDA", "Mistigris"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("credits panel is missing %q", want)
		}
	}
}

func TestEnterOpensTodaysDoor(t *testing.T) {
	today := artLine(t, "art/2025/03_DEC25.ANS")
	first := artLine(t, "art/2025/01_DEC25.ANS")
	out := runDoorProcess(t, "\rqq", "-local", "-nodetect", "-debug-date=2025-12-03")
	if !bytes.Contains(out, today) {
		t.Fatal("Enter on 3 December did not open door 3")
	}
	if !bytes.Equal(today, first) && bytes.Contains(out, first) {
		t.Fatal("Enter opened door 1 during the season")
	}
}

func TestJumpToDoorNumber(t *testing.T) {
	door12 := artLine(t, "art/2025/12_DEC25.ANS")
	out := runDoorScript(t, []string{"-local", "-nodetect", "-debug-date=2026-07-04"},
		"\r", 300*time.Millisecond, "12", 300*time.Millisecond, "qq")
	if !bytes.Contains(out, door12) {
		t.Fatal("typing 12 on a door did not open door 12")
	}
}

func TestSlideshowAdvances(t *testing.T) {
	door3 := artLine(t, "art/2025/03_DEC25.ANS")
	out := runDoorScript(t, []string{"-local", "-nodetect", "-debug-date=2026-07-04", "-slide=150ms"},
		300*time.Millisecond, "s", 900*time.Millisecond, "q", 200*time.Millisecond, "qq")
	if !bytes.Contains(out, []byte("Slideshow")) {
		t.Fatal("no slideshow note")
	}
	if !bytes.Contains(out, door3) {
		t.Fatal("slideshow did not advance to door 3")
	}
}
