package input

import (
	gd "github.com/robbiew/godoors"
	"testing"
)

func TestGoDoorsKeyTranslation(t *testing.T) {
	cases := []struct {
		ch     rune
		key    gd.Key
		wantCh rune
		want   Key
	}{
		{'q', gd.KeyNone, 'q', KeyUnknown}, {' ', gd.KeyNone, 0, KeySpace}, {'\t', gd.KeyNone, 0, KeyTab},
		{'\b', gd.KeyNone, 0, KeyBackspace}, {0x7f, gd.KeyNone, 0, KeyBackspace},
		{0, gd.KeyEnter, 0, KeyEnter}, {0, gd.KeyEsc, 0, KeyEsc}, {0, gd.KeyUp, 0, KeyArrowUp},
		{0, gd.KeyDown, 0, KeyArrowDown}, {0, gd.KeyLeft, 0, KeyArrowLeft}, {0, gd.KeyRight, 0, KeyArrowRight},
		{0, gd.KeyPageUp, 0, KeyPageUp}, {0, gd.KeyPageDown, 0, KeyPageDown}, {0, gd.KeyHome, 0, KeyHome},
		{0, gd.KeyEnd, 0, KeyEnd}, {0, gd.KeyInsert, 0, KeyInsert}, {0, gd.KeyDelete, 0, KeyDelete},
	}
	for n := 0; n < 12; n++ {
		cases = append(cases, struct {
			ch     rune
			key    gd.Key
			wantCh rune
			want   Key
		}{0, gd.KeyF1 + gd.Key(n), 0, KeyF1 + Key(n)})
	}
	for _, c := range cases {
		ch, key, err := translateKey(c.ch, c.key)
		if err != nil || ch != c.wantCh || key != c.want {
			t.Errorf("input (%q,%v): got (%q,%v,%v), want (%q,%v)", c.ch, c.key, ch, key, err, c.wantCh, c.want)
		}
	}
}
