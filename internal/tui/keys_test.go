package tui

import (
	"slices"
	"testing"
)

func TestParseKeys(t *testing.T) {
	r := func(c rune) Key { return Key{Rune: c} }
	k := func(c KeyCode) Key { return Key{Code: c} }
	tests := []struct {
		name string
		in   string
		want []Key
		rest string
	}{
		{"letters", "jkä", []Key{r('j'), r('k'), r('ä')}, ""},
		{"arrows", "\x1b[A\x1b[B\x1bOC\x1b[D", []Key{k(KeyUp), k(KeyDown), k(KeyRight), k(KeyLeft)}, ""},
		{"paging", "\x1b[5~\x1b[6~\x06\x02", []Key{k(KeyPgUp), k(KeyPgDn), k(KeyPgDn), k(KeyPgUp)}, ""},
		{"home end", "\x1b[H\x1b[1~\x1b[F\x1b[4~", []Key{k(KeyHome), k(KeyHome), k(KeyEnd), k(KeyEnd)}, ""},
		{"controls", "\r\x7f\x03", []Key{k(KeyEnter), k(KeyBackspace), k(KeyCtrlC)}, ""},
		{"lone esc", "\x1b", []Key{k(KeyEsc)}, ""},
		{"esc then letter", "\x1bq", []Key{k(KeyEsc), r('q')}, ""},
		{"unknown sequence skipped", "\x1b[99zq", []Key{r('q')}, ""},
		{"incomplete sequence", "j\x1b[5", []Key{r('j')}, "\x1b[5"},
		{"incomplete rune", "a\xc3", []Key{r('a')}, "\xc3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rest := parseKeys([]byte(tt.in))
			if !slices.Equal(got, tt.want) || string(rest) != tt.rest {
				t.Errorf("got %v rest %q, want %v rest %q", got, rest, tt.want, tt.rest)
			}
		})
	}
}
