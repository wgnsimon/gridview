package layout

import (
	"strings"
	"unicode/utf8"
)

// Column is the width and alignment of one table column.
type Column struct {
	Width int
	Right bool
}

// Formatter turns rows into aligned lines.
type Formatter struct {
	Cols []Column
	Sep  string
	// Cut shortens cells wider than their column with an ellipsis.
	Cut bool
}

const ellipsis = "…"

// AppendRow appends row without a trailing newline. The last column is not padded.
func (f *Formatter) AppendRow(dst []byte, row []string) []byte {
	for i, c := range f.Cols {
		if i > 0 {
			dst = append(dst, f.Sep...)
		}
		s := ""
		if i < len(row) {
			s = row[i]
		}
		dst = appendCell(dst, s, c, f.Cut, i < len(f.Cols)-1)
	}
	return dst
}

// AppendRule appends a separator line like "---+-----".
func (f *Formatter) AppendRule(dst []byte) []byte {
	sep := ruleSep(f.Sep)
	for i, c := range f.Cols {
		if i > 0 {
			dst = append(dst, sep...)
		}
		dst = appendRepeat(dst, '-', c.Width)
	}
	return dst
}

func ruleSep(sep string) string {
	b := []byte(sep)
	for i, c := range b {
		if c == ' ' {
			b[i] = '-'
		} else {
			b[i] = '+'
		}
	}
	return string(b)
}

func appendCell(dst []byte, s string, c Column, cut, pad bool) []byte {
	sw := Width(s)
	if sw > c.Width && cut {
		return appendCut(dst, s, c.Width)
	}
	gap := c.Width - sw
	if c.Right {
		dst = appendRepeat(dst, ' ', gap)
	}
	dst = appendClean(dst, s)
	if !c.Right && pad {
		dst = appendRepeat(dst, ' ', gap)
	}
	return dst
}

func appendCut(dst []byte, s string, w int) []byte {
	if w <= 0 {
		return dst
	}
	used := 0
	for _, r := range s {
		rw := RuneWidth(r)
		if used+rw > w-1 {
			break
		}
		dst = appendRune(dst, r)
		used += rw
	}
	dst = append(dst, ellipsis...)
	return appendRepeat(dst, ' ', w-1-used)
}

func appendClean(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] == 0x7F {
			for _, r := range s {
				dst = appendRune(dst, r)
			}
			return dst
		}
	}
	return append(dst, s...)
}

func appendRune(dst []byte, r rune) []byte {
	switch {
	case r == '\n':
		return append(dst, "↵"...)
	case r < ' ' || r == 0x7F:
		return append(dst, ' ')
	case r < utf8.RuneSelf:
		return append(dst, byte(r))
	}
	return utf8.AppendRune(dst, r)
}

var repeats = map[byte]string{' ': strings.Repeat(" ", 256), '-': strings.Repeat("-", 256)}

func appendRepeat(dst []byte, b byte, n int) []byte {
	block := repeats[b]
	for n > 0 {
		k := min(n, len(block))
		dst = append(dst, block[:k]...)
		n -= k
	}
	return dst
}
