package display

import (
	"strconv"
	"strings"
)

// sgrState tracks the colour attributes art has set (SGR sequences), so a
// window that starts partway down the art, or a row repainted on its own,
// can begin with the colours the rows above it left in effect. Art often
// sets a colour once and relies on it for several lines.
type sgrState struct {
	bold, blink, reverse bool
	fg, bg               int // 0 = default; otherwise the SGR code (30-37, 90-97 / 40-47, 100-107)
}

// apply updates the state with every SGR sequence in line.
func (s *sgrState) apply(line string) {
	for i := 0; i < len(line); {
		if line[i] != '\x1b' {
			i++
			continue
		}
		n := escapeLen(line, i)
		if seq := line[i : i+n]; len(seq) >= 3 && seq[1] == '[' && seq[len(seq)-1] == 'm' {
			s.applyParams(seq[2 : len(seq)-1])
		}
		i += n
	}
}

func (s *sgrState) applyParams(params string) {
	if params == "" {
		*s = sgrState{}
		return
	}
	fields := strings.Split(params, ";")
	for i := 0; i < len(fields); i++ {
		p, err := strconv.Atoi(fields[i])
		if err != nil {
			continue
		}
		switch {
		case p == 0:
			*s = sgrState{}
		case p == 1:
			s.bold = true
		case p == 22:
			s.bold = false
		case p == 5 || p == 6:
			s.blink = true
		case p == 25:
			s.blink = false
		case p == 7:
			s.reverse = true
		case p == 27:
			s.reverse = false
		case p >= 30 && p <= 37, p >= 90 && p <= 97:
			s.fg = p
		case p == 39:
			s.fg = 0
		case p >= 40 && p <= 47, p >= 100 && p <= 107:
			s.bg = p
		case p == 49:
			s.bg = 0
		case p == 38 || p == 48:
			// Extended colour (38;5;n or 38;2;r;g;b): not used by CP437 art; skip its arguments.
			if i+1 < len(fields) && fields[i+1] == "5" {
				i += 2
			} else if i+1 < len(fields) && fields[i+1] == "2" {
				i += 4
			}
		}
	}
}

// sequence returns an SGR sequence that recreates the state from a reset, or
// "" for the default state.
func (s sgrState) sequence() string {
	if s == (sgrState{}) {
		return ""
	}
	codes := []string{"0"}
	if s.bold {
		codes = append(codes, "1")
	}
	if s.blink {
		codes = append(codes, "5")
	}
	if s.reverse {
		codes = append(codes, "7")
	}
	if s.fg != 0 {
		codes = append(codes, strconv.Itoa(s.fg))
	}
	if s.bg != 0 {
		codes = append(codes, strconv.Itoa(s.bg))
	}
	return Esc + strings.Join(codes, ";") + "m"
}
