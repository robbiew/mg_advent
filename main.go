package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/eiannone/keyboard"
)

type User struct {
	Alias     string
	TimeLeft  time.Duration
	Emulation int
	NodeNum   int
	H         int
	W         int
	ModalH    int
	ModalW    int
}

const (
	ArtFileDir  = "art"
	WelcomeFile = "WELCOME.ANS"
	ExitFile    = "GOODBYE.ANS"
	DateFormat  = "_DEC23.ANS"
)

var (
	DropPath     string
	today        int
	timeOut      time.Duration
	localDisplay bool
	u            User // Global User object
)

func init() {
	timeOut = 1 * time.Minute
	pathPtr := flag.String("path", "", "path to door32.sys file (optional if --local is set)")
	localDisplayPtr := flag.Bool("local", false, "use local UTF-8 display instead of CP437")
	flag.Parse()

	localDisplay = *localDisplayPtr // Set the global variable

	if localDisplay {
		// Set default values when --local is used
		u = User{
			Alias:     "SysOp",
			TimeLeft:  120 * time.Minute,
			Emulation: 1,
			NodeNum:   1,
			H:         25,
			W:         80,
			ModalH:    25,
			ModalW:    80,
		}
	} else {
		// Check for required --path argument if --local is not set
		if *pathPtr == "" {
			fmt.Fprintln(os.Stderr, "missing required -path argument")
			os.Exit(2)
		}
		DropPath = *pathPtr
	}
}

func getDay() {
	_, month, day := time.Now().Date()
	today = 25
	if month == time.December {
		today = day
	}
}

func displayArt(today int) {
	artFileName := filepath.Join(ArtFileDir, strconv.Itoa(today)+DateFormat)
	displayAnsiFile(artFileName)
}

func ReadAnsiFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func displayAnsiFile(filePath string) {
	content, err := ReadAnsiFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file %s: %v", filePath, err)
	}
	ClearScreen()
	PrintAnsi(content, 0, localDisplay)
}

func main() {
	// Get door32.sys as user object
	// Using TimeLeft, H, W, Emulation
	u := Initialize(DropPath)
	getDay()

	// Exit if no ANSI capabilities (sorry!)
	if u.Emulation != 1 {
		fmt.Println("Sorry, ANSI is required to use this...")
		time.Sleep(time.Duration(2) * time.Second)
		os.Exit(0)
	}

	if err := keyboard.Open(); err != nil {
		fmt.Println(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	// start idle and max timers
	timerManager := NewTimerManager(timeOut, u.TimeLeft)
	timerManager.StartIdleTimer()
	timerManager.StartMaxTimer()

	CursorHide()
	ClearScreen()

	displayAnsiFile(filepath.Join(ArtFileDir, WelcomeFile))
	Pause(u.H-2, u.W)
	displayArt(today)

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		timerManager.ResetIdleTimer() // Resets the idle timer on key press

		if key == keyboard.KeyArrowLeft || key == keyboard.KeyArrowRight {
			updatedDay := today

			if key == keyboard.KeyArrowLeft && today > 1 {
				updatedDay = today - 1
			} else if key == keyboard.KeyArrowRight && today < 25 {
				updatedDay = today + 1
			}

			if updatedDay != today {
				today = updatedDay
				ClearScreen()
				fmt.Print(Reset)
				displayArt(today)
			}
		} else if string(char) == "q" || string(char) == "Q" || key == keyboard.KeyEsc {
			defer timerManager.StopIdleTimer()
			defer timerManager.StopMaxTimer()
			displayAnsiFile(filepath.Join(ArtFileDir, ExitFile))
			Pause(u.H-2, u.W)
			CursorShow()
			return
		}
	}
}
