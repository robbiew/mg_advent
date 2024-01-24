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
	DropPath string
	today    int
	timeOut  time.Duration
)

func init() {
	timeOut = 1 * time.Minute
	pathPtr := flag.String("path", "", "path to door32.sys file")
	required := []string{"path"}

	flag.Parse()

	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			fmt.Fprintf(os.Stderr, "missing path to door32.sys directory: -%s \n", req)
			os.Exit(2)
		}
	}
	DropPath = *pathPtr
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
	PrintAnsi(content, 0)
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
