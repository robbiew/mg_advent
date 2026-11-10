package validation

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestValidateArtFiles(t *testing.T) {
	common := fstest.MapFS{
		"art/common/MISSING.ANS": {},
		"art/common/NOTYET.ANS":  {},
	}
	withYear := func(files ...string) fstest.MapFS {
		fs := fstest.MapFS{}
		for k, v := range common {
			fs[k] = v
		}
		for _, f := range files {
			fs["art/2026/"+f] = &fstest.MapFile{}
		}
		return fs
	}

	// A partly delivered year (no screens, one day) is still valid.
	if err := NewValidator(withYear("01_DEC26.ANS"), "art").ValidateArtFiles(2026); err != nil {
		t.Fatalf("partial year rejected: %v", err)
	}
	if err := NewValidator(withYear(), "art").ValidateArtFiles(2027); err == nil {
		t.Fatal("a year with no directory should fail")
	}
	noCommon := fstest.MapFS{"art/2026/WELCOME.ANS": {}}
	err := NewValidator(noCommon, "art").ValidateArtFiles(2026)
	if err == nil || !strings.Contains(err.Error(), "MISSING.ANS") {
		t.Fatalf("missing common fallbacks should fail, got %v", err)
	}
}

func TestValidateTerminal(t *testing.T) {
	v := NewValidator(fstest.MapFS{}, "art")
	if v.ValidateTerminalSize(80, 24) != nil || v.ValidateTerminalSize(79, 25) == nil || v.ValidateTerminalSize(80, 23) == nil {
		t.Fatal("terminal size bounds wrong")
	}
	if v.ValidateEmulation(1) != nil || v.ValidateEmulation(0) == nil {
		t.Fatal("emulation check wrong")
	}
}
