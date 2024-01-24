package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/eiannone/keyboard"
)

// TimerManager manages timers for idle and max timeout
type TimerManager struct {
	idleTimer    *time.Timer
	maxTimer     *time.Timer
	idleDuration time.Duration
	maxDuration  time.Duration
	lock         sync.Mutex
}

const (
	Esc = "\u001B["
	Osc = "\u001B]"
	Bel = "\u0007"
)

// Common fonts, supported by SyncTerm
const (
	Mosoul          = Esc + "0;38 D"
	Potnoodle       = Esc + "0;37 D"
	Microknight     = Esc + "0;41 D"
	Microknightplus = Esc + "0;39 D"
	Topaz           = Esc + "0;42 D"
	Topazplus       = Esc + "0;40 D"
	Ibm             = Esc + "0;0 D"
	Ibmthin         = Esc + "0;26 D"
)

// Symbols
var (
	Heart        = string([]rune{'\u0003'})
	ArrowUpDown  = string([]rune{'\u0017'})
	ArrowUp      = string([]rune{'\u0018'})
	ArrowDown    = string([]rune{'\u0019'})
	ArrowDownFat = string([]rune{'\u001F'})
	ArrowRight   = string([]rune{'\u0010'})
	ArrowLeft    = string([]rune{'\u0011'})
	Block        = string([]rune{'\u0219'})

	modalH int // in case height is odd
	modalW int // in case width is odd
)

// Common ANSI escapes sequences. This is not a complete list.
const (
	CursorBackward = Esc + "D"
	CursorPrevLine = Esc + "F"
	CursorLeft     = Esc + "G"
	CursorTop      = Esc + "d"
	CursorTopLeft  = Esc + "H"

	CursorBlinkEnable  = Esc + "?12h"
	CursorBlinkDisable = Esc + "?12I"

	ScrollUp   = Esc + "S"
	ScrollDown = Esc + "T"

	TextInsertChar = Esc + "@"
	TextDeleteChar = Esc + "P"
	TextEraseChar  = Esc + "X"
	TextInsertLine = Esc + "L"
	TextDeleteLine = Esc + "M"

	EraseRight  = Esc + "K"
	EraseLeft   = Esc + "1K"
	EraseLine   = Esc + "2K"
	EraseDown   = Esc + "J"
	EraseUp     = Esc + "1J"
	EraseScreen = Esc + "2J"

	Black     = Esc + "30m"
	Red       = Esc + "31m"
	Green     = Esc + "32m"
	Yellow    = Esc + "33m"
	Blue      = Esc + "34m"
	Magenta   = Esc + "35m"
	Cyan      = Esc + "36m"
	White     = Esc + "37m"
	BlackHi   = Esc + "30;1m"
	RedHi     = Esc + "31;1m"
	GreenHi   = Esc + "32;1m"
	YellowHi  = Esc + "33;1m"
	BlueHi    = Esc + "34;1m"
	MagentaHi = Esc + "35;1m"
	CyanHi    = Esc + "36;1m"
	WhiteHi   = Esc + "37;1m"

	BgBlack     = Esc + "40m"
	BgRed       = Esc + "41m"
	BgGreen     = Esc + "42m"
	BgYellow    = Esc + "43m"
	BgBlue      = Esc + "44m"
	BgMagenta   = Esc + "45m"
	BgCyan      = Esc + "46m"
	BgWhite     = Esc + "47m"
	BgBlackHi   = Esc + "40;1m"
	BgRedHi     = Esc + "41;1m"
	BgGreenHi   = Esc + "42;1m"
	BgYellowHi  = Esc + "43;1m"
	BgBlueHi    = Esc + "44;1m"
	BgMagentaHi = Esc + "45;1m"
	BgCyanHi    = Esc + "46;1m"
	BgWhiteHi   = Esc + "47;1m"

	Reset = Esc + "0m"
)

var Idle int

// Get info from the Drop File, h, w
func Initialize(path string) User {

	alias, timeLeft, emulation, nodeNum := DropFileData(path)
	h, w := GetTermSize()

	if h%2 == 0 {
		modalH = h
	} else {
		modalH = h - 1
	}

	if w%2 == 0 {
		modalW = w
	} else {
		modalW = w - 1
	}

	timeLeftDuration := time.Duration(timeLeft) * time.Minute

	u := User{
		Alias:     alias,
		TimeLeft:  timeLeftDuration,
		Emulation: emulation,
		NodeNum:   nodeNum,
		H:         h,
		W:         w,
		ModalH:    modalH,
		ModalW:    modalW,
	}
	return u
}

// Continue Y/N
func Continue() bool {

	char, key, err := keyboard.GetKey()
	if err != nil {
		panic(err)
	}
	var x bool
	if string(char) == "Y" || string(char) == "y" || key == keyboard.KeyEnter {
		x = true
	}
	if string(char) == "N" || string(char) == "n" || key == keyboard.KeyEsc {
		x = false
	}
	return x
}

func Pause(h int, w int) {
	MoveCursor(0, h)

	CenterText("Press any key to continue...", w)

	_, _, err := keyboard.GetKey()
	if err != nil {
		panic(err)
	}
}

// Move cursor to X, Y location
func MoveCursor(x int, y int) {
	fmt.Printf(Esc+"%d;%df", y, x)
}

// Erase the screen
func ClearScreen() {
	fmt.Println(EraseScreen)
	MoveCursor(0, 0)
}

// Move the cursor n cells to up.
func CursorUp(n int) {
	fmt.Printf(Esc+"%dA", n)
}

// Move the cursor n cells to down.
func CursorDown(n int) {
	fmt.Printf(Esc+"%dB", n)
}

// Move the cursor n cells to right.
func CursorForward(n int) {
	fmt.Printf(Esc+"%dC", n)
}

// Move the cursor n cells to left.
func CursorBack(n int) {
	fmt.Printf(Esc+"%dD", n)
}

// Move cursor to beginning of the line n lines down.
func CursorNextLine(n int) {
	fmt.Printf(Esc+"%dE", n)
}

// Move cursor to beginning of the line n lines up.
func CursorPreviousLine(n int) {
	fmt.Printf(Esc+"%dF", n)
}

// Move cursor horizontally to x.
func CursorHorizontalAbsolute(x int) {
	fmt.Printf(Esc+"%dG", x)
}

// Show the cursor.
func CursorShow() {
	fmt.Print(Esc + "?25h")
}

// Hide the cursor.
func CursorHide() {
	fmt.Print(Esc + "?25l")
}

// Save the screen.
func SaveScreen() {
	fmt.Print(Esc + "?47h")
}

// Restore the saved screen.
func RestoreScreen() {
	fmt.Print(Esc + "?47l")
}

func DropFileData(path string) (string, int, int, int) {
	// path needs to include trailing slash!
	var dropAlias string
	var dropTimeLeft string
	var dropEmulation string
	var nodeNum string

	file, err := os.Open(strings.ToLower(path + "door32.sys"))
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string

	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	file.Close()

	count := 0
	for _, line := range text {
		if count == 6 {
			dropAlias = line
		}
		if count == 8 {
			dropTimeLeft = line
		}
		if count == 9 {
			dropEmulation = line
		}
		if count == 10 {
			nodeNum = line
		}
		if count == 11 {
			break
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	timeInt, err := strconv.Atoi(dropTimeLeft) // return as int
	if err != nil {
		log.Fatal(err)
	}

	emuInt, err := strconv.Atoi(dropEmulation) // return as int
	if err != nil {
		log.Fatal(err)
	}
	nodeInt, err := strconv.Atoi(nodeNum) // return as int
	if err != nil {
		log.Fatal(err)
	}

	return dropAlias, timeInt, emuInt, nodeInt
}

/*
Get the terminal size
- Send a cursor position that we know is way too large
- Terminal sends back the largest row + col size
- Read in the result
*/
func GetTermSize() (int, int) {
	// Set the terminal to raw mode so we aren't waiting for CLRF rom user (to be undone with `-raw`)
	rawMode := exec.Command("/bin/stty", "raw")
	rawMode.Stdin = os.Stdin
	_ = rawMode.Run()

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "\033[999;999f") // larger than any known term size
	fmt.Fprintf(os.Stdout, "\033[6n")       // ansi escape code for reporting cursor location
	text, _ := reader.ReadString('R')

	// Set the terminal back from raw mode to 'cooked'
	rawModeOff := exec.Command("/bin/stty", "-raw")
	rawModeOff.Stdin = os.Stdin
	_ = rawModeOff.Run()
	rawModeOff.Wait()

	// check for the desired output
	if strings.Contains(string(text), ";") {
		re := regexp.MustCompile(`\d+;\d+`)
		line := re.FindString(string(text))

		s := strings.Split(line, ";")
		sh, sw := s[0], s[1]

		ih, err := strconv.Atoi(sh)
		if err != nil {
			// handle error
			fmt.Println(err)
			os.Exit(2)
		}

		iw, err := strconv.Atoi(sw)
		if err != nil {
			// handle error
			fmt.Println(err)
			os.Exit(2)
		}
		h := ih
		w := iw

		ClearScreen()

		return h, w

	} else {
		// couldn't detect, so let's just set 80 x 25 to be safe
		h := 80
		w := 25

		return h, w
	}

}

// Print ANSI art with a delay between lines
func PrintAnsi(artContent string, delay int) {

	noSauce := TrimStringFromSauce(artContent) // strip off the SAUCE metadata
	lines := strings.Split(noSauce, "\r\n")
	for i, line := range lines {
		if i < len(lines)-1 && i != 24 { // Check for the 25th line (index 24)
			fmt.Println(line) // Print with a newline
		} else {
			fmt.Print(line) // Print without a newline (for the 25th line and the last line of the art)
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// TrimStringFromSauce trims SAUCE metadata from a string.
func TrimStringFromSauce(s string) string {
	return trimMetadata(s, "COMNT", "SAUCE00")
}

// trimMetadata trims metadata based on delimiters.
func trimMetadata(s string, delimiters ...string) string {
	for _, delimiter := range delimiters {
		if idx := strings.Index(s, delimiter); idx != -1 {
			return trimLastChar(s[:idx])
		}
	}
	return s
}

// trimLastChar trims the last character from a string.
func trimLastChar(s string) string {
	if len(s) > 0 {
		_, size := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-size]
	}
	return s
}

// Print ANSI art at an X, Y location
func PrintAnsiLoc(artfile string, x int, y int) {
	yLoc := y

	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(yLoc)+";"+strconv.Itoa(x)+"f"+s.Text())
		yLoc++
	}
}

// Print text at an X, Y location
func PrintStringLoc(text string, x int, y int) {
	fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(y)+";"+strconv.Itoa(x)+"f"+text)
}

// CenterText horizontally centers some text
func CenterText(s string, w int) {
	padding := (w - len(s)) / 2
	if padding < 0 {
		padding = 0
	}
	// Pad the left side of the string with spaces to center the text
	fmt.Fprintf(os.Stdout, Cyan+"%[1]*s\n", -w, fmt.Sprintf("%[1]*s"+Reset, padding+len(s), s))
}

// Horizontally and Vertically center some text.
func AbsCenterText(s string, l int, c string) {
	centerY := modalH / 2
	halfLen := l / 2
	centerX := (modalW - modalW/2) - halfLen
	MoveCursor(centerX, centerY)
	fmt.Fprintf(os.Stdout, WhiteHi+c+s+Reset)
	result := Continue()
	if result {
		fmt.Fprintf(os.Stdout, BgCyan+CyanHi+" Yes"+Reset)
		time.Sleep(1 * time.Second)
	}
	if !result {
		fmt.Fprintf(os.Stdout, BgCyan+CyanHi+" No"+Reset)
		time.Sleep(1 * time.Second)
	}
}

func AbsCenterArt(artfile string, l int) {
	artY := (modalH / 2) - 2
	artLen := l / 2
	artX := (modalW - modalW/2) - artLen

	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(artY)+";"+strconv.Itoa(artX)+"f")
		fmt.Println(s.Text())
		artY++
	}
}

// NewTimerManager creates a new TimerManager with specified durations
func NewTimerManager(idleDuration, maxDuration time.Duration) *TimerManager {
	return &TimerManager{
		idleDuration: idleDuration,
		maxDuration:  maxDuration,
	}
}

// StartIdleTimer starts or resets the idle timer
func (tm *TimerManager) StartIdleTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.idleTimer != nil {
		tm.idleTimer.Stop()
	}

	tm.idleTimer = time.AfterFunc(tm.idleDuration, func() {
		fmt.Println("\nYou've been idle for too long... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})
}

// StopIdleTimer stops the idle timer
func (tm *TimerManager) StopIdleTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.idleTimer != nil {
		tm.idleTimer.Stop()
	}
}

// StartMaxTimer starts the max timeout timer
func (tm *TimerManager) StartMaxTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.maxTimer != nil {
		tm.maxTimer.Stop()
	}

	tm.maxTimer = time.AfterFunc(tm.maxDuration, func() {
		fmt.Println("\nMax time exceeded... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})

}

// StopMaxTimer stops the max timeout timer
func (tm *TimerManager) StopMaxTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.maxTimer != nil {
		tm.maxTimer.Stop()
	}
}

// ResetTimers resets both idle and max timers
func (tm *TimerManager) ResetTimers() {
	tm.StartIdleTimer()
	tm.StartMaxTimer()
}

// ResetIdleTimer resets idle timers
func (tm *TimerManager) ResetIdleTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.idleTimer != nil {
		tm.idleTimer.Stop()
	}

	tm.idleTimer = time.AfterFunc(tm.idleDuration, func() {
		fmt.Println("\nYou've been idle for too long... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})
}

// ResetMaxTimer resets max timers
func (tm *TimerManager) ResetMaxTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.maxTimer != nil {
		tm.maxTimer.Stop()
	}

	tm.maxTimer = time.AfterFunc(tm.maxDuration, func() {
		fmt.Println("\nMax time exceeded... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})
}
