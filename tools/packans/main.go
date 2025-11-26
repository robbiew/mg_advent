package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var missingAnsData []byte

type IndexEntry struct {
	Offset uint32
	Size   uint32
}

func main() {
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "Usage: packans <output.dat> <screen1.ans> [screen2.ans] ...")
		fmt.Fprintln(os.Stderr, "Packs ANS files into indexed DAT format in the order given")
		os.Exit(1)
	}

	outputFile := flag.Arg(0)
	inputFiles := flag.Args()[1:]

	// Load MISSING.ANS for fallback
	missingPath := "../internal/embedded/art/common/MISSING.ANS"
	var err error
	missingAnsData, err = os.ReadFile(missingPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Cannot load %s\n", missingPath)
		missingAnsData = []byte{}
	} else {
		missingAnsData = stripSAUCE(missingAnsData)
		fmt.Printf("Loaded MISSING.ANS (%d bytes)\n", len(missingAnsData))
	}

	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", outputFile, err)
		os.Exit(1)
	}
	defer out.Close()

	numScreens := len(inputFiles)
	indexSize := 4 + (numScreens * 8)

	// Write number of screens
	if err := binary.Write(out, binary.LittleEndian, uint32(numScreens)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing header: %v\n", err)
		os.Exit(1)
	}

	// Build index and data
	index := make([]IndexEntry, numScreens)
	allData := make([][]byte, numScreens)
	currentOffset := uint32(indexSize)

	for i, inputFile := range inputFiles {
		var data []byte
		baseName := filepath.Base(inputFile)

		d, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Printf("Screen %2d:  MISSING  (was %s)\n", i, baseName)
			// Make fresh copy with date overlay for missing daily art only
			data = make([]byte, len(missingAnsData))
			copy(data, missingAnsData)

			// Add date overlay ONLY for missing daily art files
			if isDailyArt(baseName) {
				data = addDateOverlay(data, baseName)
			}
		} else {
			// Existing art - just strip SAUCE, NO date overlay
			data = stripSAUCE(d)
			fmt.Printf("Screen %2d: %s (%d bytes)\n", i, baseName, len(data))
		}

		allData[i] = data
		index[i] = IndexEntry{
			Offset: currentOffset,
			Size:   uint32(len(data)),
		}
		currentOffset += uint32(len(data))
	}

	// Write index
	for _, entry := range index {
		binary.Write(out, binary.LittleEndian, entry.Offset)
		binary.Write(out, binary.LittleEndian, entry.Size)
	}

	// Write data
	for _, data := range allData {
		out.Write(data)
	}

	fmt.Printf("\nCreated %s: %d screens, %d bytes\n", outputFile, numScreens, currentOffset)
}

func stripSAUCE(data []byte) []byte {
	markers := [][]byte{[]byte("SAUCE00"), []byte("COMNT")}
	minIdx := len(data)

	for _, marker := range markers {
		for i := 0; i+len(marker) <= len(data); i++ {
			match := true
			for j := 0; j < len(marker); j++ {
				if data[i+j] != marker[j] {
					match = false
					break
				}
			}
			if match && i < minIdx {
				if i > 0 && data[i-1] == 0x1A {
					minIdx = i - 1
				} else {
					minIdx = i
				}
				break
			}
		}
	}

	if minIdx < len(data) {
		return data[:minIdx]
	}
	return data
}

func isDailyArt(filename string) bool {
	// Daily art files: N_DECYY.ANS or NN_DECYY.ANS
	base := filepath.Base(filename)
	if len(base) < 10 {
		return false
	}
	if base == "WELCOME.ANS" || base == "INFOFILE.ANS" || base == "MEMBERS.ANS" {
		return false
	}
	// Check for _DEC pattern
	for i := 0; i < len(base)-6; i++ {
		if base[i] == '_' && i+7 < len(base) && base[i+1:i+4] == "DEC" {
			return true
		}
	}
	return false
}

func addDateOverlay(data []byte, filename string) []byte {
	var day, year int
	fmt.Sscanf(filename, "%d_DEC%d", &day, &year)

	if day > 0 && day <= 25 && year >= 0 {
		dateStr := fmt.Sprintf("12/%d/%02d", day, year)
		overlay := fmt.Sprintf("\x1b[25;70H\x1b[97;40m%s\x1b[0m", dateStr)
		return append(data, []byte(overlay)...)
	}

	return data
}
