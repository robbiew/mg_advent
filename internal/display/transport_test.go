package display

import (
	"bytes"
	"fmt"
	"testing"
	"testing/fstest"
)

func TestTransportPreservesArtEncoding(t *testing.T) {
	for _, mode := range []DisplayMode{ModeCP437Raw, ModeCP437} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			content := []byte{0xdb, 0xc4, '\r', '\n'}
			fs := fstest.MapFS{"art/test.ans": {Data: content}}
			engine := NewDisplayEngine(DisplayConfig{Mode: mode, Width: 80, Height: 25}, fs)
			var out bytes.Buffer
			engine.SetBBSConnection(&out)
			if err := engine.Display("art/test.ans", User{W: 80, H: 25}); err != nil {
				t.Fatal(err)
			}
			want := []byte{0xdb, 0xc4}
			if mode == ModeCP437 {
				want = []byte("█─")
			}
			if !bytes.Contains(out.Bytes(), want) {
				t.Fatalf("missing %x in output %x", want, out.Bytes())
			}
		})
	}
}
