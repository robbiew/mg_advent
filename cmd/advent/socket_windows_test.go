//go:build windows

package main

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Run the actual entry point with an inherited native socket, rather than a
// Go net.Conn substituted into the app. Stdout must not receive caller output.
func TestWindowsDoorInheritedSocket(t *testing.T) {
	for _, noDetect := range []bool{true, false} {
		t.Run(fmt.Sprint(noDetect), func(t *testing.T) {
			var data windows.WSAData
			if err := windows.WSAStartup(0x202, &data); err != nil {
				t.Fatal(err)
			}
			defer windows.WSACleanup()
			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			h, err := windows.Socket(windows.AF_INET, windows.SOCK_STREAM, windows.IPPROTO_TCP)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if h != windows.InvalidHandle {
					windows.Closesocket(h)
				}
			}()
			if err := windows.Connect(h, &windows.SockaddrInet4{Port: listener.Addr().(*net.TCPAddr).Port, Addr: [4]byte{127, 0, 0, 1}}); err != nil {
				t.Fatal(err)
			}
			peer, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			defer peer.Close()
			peer.SetDeadline(time.Now().Add(10 * time.Second))
			if err := windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			path := filepath.Join(dir, "door32.sys")
			content := fmt.Sprintf("2\n%d\n57600\nTest BBS\n1\nReal Name\nCaller\n255\n42\n1\n3\n", h)
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			keys := "qq"
			if !noDetect {
				keys = "q\x1b[25;80R\x1b[?1;2cq"
			}
			if _, err := peer.Write([]byte(keys)); err != nil {
				t.Fatal(err)
			}
			type result struct {
				data []byte
				err  error
			}
			output := make(chan result, 1)
			go func() { b, err := io.ReadAll(peer); output <- result{b, err} }()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestDoorProcess$", "-path", path, "-debug-disable-date", "-noice"}
			if noDetect {
				args = append(args, "-nodetect")
			}
			cmd := exec.CommandContext(ctx, os.Args[0], args...)
			cmd.Env = append(os.Environ(), "MG_ADVENT_TEST_PROCESS=1")
			cmd.SysProcAttr = &syscall.SysProcAttr{AdditionalInheritedHandles: []syscall.Handle{syscall.Handle(h)}}
			cmd.Stdin = strings.NewReader("wrong-input")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("child: %v, stderr: %s", err, stderr.String())
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("caller I/O leaked to console: stdout %d, stderr %s", stdout.Len(), stderr.String())
			}
			// The parent's descriptor must still work after GoDoors closes the child.
			text := []byte("parent-resumed")
			buf := windows.WSABuf{Len: uint32(len(text)), Buf: &text[0]}
			var n uint32
			if err := windows.WSASend(h, &buf, 1, &n, 0, nil, nil); err != nil || n != uint32(len(text)) {
				t.Fatalf("parent send: %d %v", n, err)
			}
			if err := windows.Closesocket(h); err != nil {
				t.Fatal(err)
			}
			h = windows.InvalidHandle
			got := <-output
			if got.err != nil {
				t.Fatal(got.err)
			}
			if !bytes.HasSuffix(got.data, text) {
				t.Fatal("parent connection did not resume")
			}
			if len(got.data) < 100 {
				t.Fatal("door did not draw art")
			}
			probed := bytes.Contains(got.data, []byte("\x1b[6n"))
			if probed == noDetect {
				t.Fatalf("probe=%v, noDetect=%v", probed, noDetect)
			}
		})
	}
}
