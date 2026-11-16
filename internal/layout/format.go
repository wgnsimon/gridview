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
	return f.AppendMarked(dst, row, -1, "", "")
}

// AppendMarked is AppendRow with column mark wrapped in the on and off sequences.
func (f *Formatter) AppendMarked(dst []byte, row []string, mark int, on, off string) []byte {
	for i, c := range f.Cols {
		if i > 0 {
			dst = append(dst, f.Sep...)
		}
		s := ""
		if i < len(row) {
			s = row[i]
		}
		if i == mark {
			dst = append(dst, on...)
		}
		dst = appendCell(dst, s, c, f.Cut, i < len(f.Cols)-1)
		if i == mark {
			dst = append(dst, off...)
		}
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
		dst = appendRepeat(dst, dashes, c.Width)
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
	sw, clean := measure(s)
	if sw > c.Width && cut {
		if clean && sw == len(s) && c.Width > 0 {
			return append(append(dst, s[:c.Width-1]...), ellipsis...)
		}
		return appendCut(dst, s, c.Width)
	}
	gap := c.Width - sw
	if c.Right {
		dst = appendRepeat(dst, spaces, gap)
	}
	if clean {
		dst = append(dst, s...)
	} else {
		for _, r := range s {
			dst = appendRune(dst, r)
		}
	}
	if !c.Right && pad {
		dst = appendRepeat(dst, spaces, gap)
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
	return appendRepeat(dst, spaces, w-1-used)
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

var (
	spaces = strings.Repeat(" ", 256)
	dashes = strings.Repeat("-", 256)
)

func appendRepeat(dst []byte, block string, n int) []byte {
	for n > 0 {
		k := min(n, len(block))
		dst = append(dst, block[:k]...)
		n -= k
	}
	return dst
}
