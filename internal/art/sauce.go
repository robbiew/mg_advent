package art

import (
	"io/fs"
	"strings"
	"time"
)

// Credits is the authorship recorded in an art file's SAUCE record
// (http://www.acid.org/info/sauce/sauce.htm). Text is CP437.
type Credits struct {
	Title, Artist, Group string
	Date                 time.Time // zero when not recorded
}

// ReadCredits returns the SAUCE credits of data, or ok=false when it has no
// SAUCE record. Like gd.TrimStringFromSauce, it only accepts a record
// anchored at the very end of the file.
func ReadCredits(data []byte) (c Credits, ok bool) {
	const recordLen = 128
	if len(data) < recordLen {
		return Credits{}, false
	}
	rec := data[len(data)-recordLen:]
	if string(rec[:7]) != "SAUCE00" {
		return Credits{}, false
	}
	field := func(off, width int) string {
		return strings.TrimRight(string(rec[off:off+width]), " \x00")
	}
	c = Credits{Title: field(7, 35), Artist: field(42, 20), Group: field(62, 20)}
	if d, err := time.Parse("20060102", field(82, 8)); err == nil {
		c.Date = d
	}
	return c, true
}

// Credits reads the SAUCE credits of the art file at path.
func (m *Manager) Credits(path string) (Credits, bool) {
	data, err := fs.ReadFile(m.fs, path)
	if err != nil {
		return Credits{}, false
	}
	return ReadCredits(data)
}
