package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/robbiew/advent/internal/art"
	gd "github.com/robbiew/godoors"
)

var escapes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestPlaceholdersLookLikeRealArt(t *testing.T) {
	root := t.TempDir()
	for _, y := range []string{"2023", "2024", "2025"} {
		os.Mkdir(filepath.Join(root, y), 0o755)
	}
	date := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	created, _, err := generate(root, 2026, false, date)
	if err != nil {
		t.Fatal(err)
	}
	if created != 30 {
		t.Fatalf("created %d files, want 30 (5 screens + 25 days)", created)
	}

	for _, s := range screensFor(2026) {
		data, err := os.ReadFile(filepath.Join(root, "2026", s.file))
		if err != nil {
			t.Fatal(err)
		}
		if !art.IsPlaceholder(data) {
			t.Errorf("%s: no placeholder marker", s.file)
		}
		body := gd.TrimStringFromSauce(string(data))
		if len(body) != len(data)-129 {
			t.Errorf("%s: SAUCE record not recognised", s.file)
		}
		lines := strings.Split(body, "\r\n")
		if !s.tall && len(lines) != rows {
			t.Errorf("%s: %d rows, want %d", s.file, len(lines), rows)
		}
		if s.tall && len(lines) <= rows {
			t.Errorf("%s: %d rows; should be taller than the screen to scroll", s.file, len(lines))
		}
		for i, l := range lines {
			if w := len(escapes.ReplaceAllString(l, "")); w > cols {
				t.Errorf("%s line %d: %d columns, max %d", s.file, i+1, w, cols)
			}
		}
	}

	welcome, _ := os.ReadFile(filepath.Join(root, "2026", "WELCOME.ANS"))
	for _, want := range []string{"1  2023 ADVENT", "2  2024 ADVENT", "3  2025 ADVENT"} {
		if !strings.Contains(string(welcome), want) {
			t.Errorf("welcome menu is missing %q", want)
		}
	}
}

func TestKeepsDeliveredArt(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "2026")
	os.MkdirAll(dir, 0o755)
	real := []byte("real art from the artist")
	os.WriteFile(filepath.Join(dir, "WELCOME.ANS"), real, 0o644)

	created, skipped, err := generate(root, 2026, false, time.Now())
	if err != nil || created != 29 || skipped != 1 {
		t.Fatalf("created %d, skipped %d, err %v", created, skipped, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "WELCOME.ANS")); string(got) != string(real) {
		t.Fatal("existing art was overwritten")
	}
}
