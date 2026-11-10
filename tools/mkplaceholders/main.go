// Command mkplaceholders writes placeholder art for an advent year, so the
// door can be set up and tested before the real pack arrives.
//
//	go run ./tools/mkplaceholders -year 2026
//
// It creates every file the door looks for in art/<year>/ (welcome, comeback
// and goodbye screens, info file, member list and 25 daily doors). Existing
// files are never overwritten unless -force is given, so it is safe to rerun
// after some real art has been dropped in. Every placeholder carries
// art.PlaceholderMarker; the door logs a warning when it shows one and the
// release scripts refuse to package them.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"
)

func main() {
	year := flag.Int("year", 0, "advent year to create placeholders for (required)")
	artRoot := flag.String("art", filepath.Join("internal", "embedded", "art"), "art directory holding one folder per year")
	force := flag.Bool("force", false, "overwrite existing files")
	flag.Parse()
	if *year < 2000 || *year > 2099 {
		fmt.Fprintln(os.Stderr, "usage: mkplaceholders -year YYYY [-art dir] [-force]")
		os.Exit(2)
	}
	created, skipped, err := generate(*artRoot, *year, *force, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkplaceholders:", err)
		os.Exit(1)
	}
	fmt.Printf("art/%d: %d placeholder files written, %d existing files kept\n", *year, created, skipped)
}

// generate writes year's placeholders under artRoot/year.
func generate(artRoot string, year int, force bool, date time.Time) (created, skipped int, err error) {
	earlier, err := earlierYears(artRoot, year)
	if err != nil {
		return 0, 0, err
	}
	dir := filepath.Join(artRoot, strconv.Itoa(year))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, 0, err
	}
	for _, s := range screensFor(year) {
		path := filepath.Join(dir, s.file)
		if !force {
			if _, err := os.Stat(path); err == nil {
				skipped++
				continue
			} else if !errors.Is(err, fs.ErrNotExist) {
				return created, skipped, err
			}
		}
		if err := os.WriteFile(path, render(s, year, earlier, date), 0o644); err != nil {
			return created, skipped, err
		}
		created++
	}
	return created, skipped, nil
}

// earlierYears lists installed year folders before year, oldest first; the
// welcome menu shows them on keys 1, 2, 3... as the door numbers them.
func earlierYears(artRoot string, year int) ([]int, error) {
	entries, err := os.ReadDir(artRoot)
	if err != nil {
		return nil, err
	}
	var years []int
	for _, e := range entries {
		if y, err := strconv.Atoi(e.Name()); err == nil && e.IsDir() && len(e.Name()) == 4 && y >= 2000 && y < year {
			years = append(years, y)
		}
	}
	slices.Sort(years)
	return years, nil
}
