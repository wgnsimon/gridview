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
	// Border draws a box-drawing frame around rows and rules.
	// Sep should then be " │ ".
	Border bool
}

// BorderWidth is the width the frame adds to a line.
const BorderWidth = 4

const ellipsis = "…"

// AppendRow appends row without a trailing newline. The last column is not padded.
func (f *Formatter) AppendRow(dst []byte, row []string) []byte {
	return f.AppendMarked(dst, row, -1, "", "")
}

// AppendMarked is AppendRow with column mark wrapped in the on and off sequences.
func (f *Formatter) AppendMarked(dst []byte, row []string, mark int, on, off string) []byte {
	if f.Border {
		dst = append(dst, "│ "...)
	}
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
		dst = appendCell(dst, s, c, f.Cut, f.Border || i < len(f.Cols)-1)
		if i == mark {
			dst = append(dst, off...)
		}
	}
	if f.Border {
		dst = append(dst, " │"...)
	}
	return dst
}

// AppendRule appends the line under the header: "---+-----" for an ASCII
// Sep, "────┼─────" for " │ ", and "├────┼─────┤" with Border.
func (f *Formatter) AppendRule(dst []byte) []byte {
	if f.Border {
		return f.appendLine(dst, "├─", "─┼─", "─┤")
	}
	return f.appendLine(dst, "", ruleSep(f.Sep), "")
}

// AppendTop appends the top of the frame, "┌────┬─────┐".
func (f *Formatter) AppendTop(dst []byte) []byte {
	return f.appendLine(dst, "┌─", "─┬─", "─┐")
}

// AppendBottom appends the bottom of the frame, "└────┴─────┘".
func (f *Formatter) AppendBottom(dst []byte) []byte {
	return f.appendLine(dst, "└─", "─┴─", "─┘")
}

func (f *Formatter) appendLine(dst []byte, left, cross, right string) []byte {
	fill := dashes
	if f.Border || strings.Contains(f.Sep, "│") {
		fill = lines
	}
	dst = append(dst, left...)
	for i, c := range f.Cols {
		if i > 0 {
			dst = append(dst, cross...)
		}
		dst = appendRepeat(dst, fill, c.Width)
	}
	return append(dst, right...)
}

// ruleSep turns a separator like " | " into "-+-" and " │ " into "─┼─".
func ruleSep(sep string) string {
	b := make([]byte, 0, len(sep)*3)
	for _, r := range sep {
		switch r {
		case ' ':
			if strings.Contains(sep, "│") {
				b = append(b, "─"...)
			} else {
				b = append(b, '-')
			}
		case '│':
			b = append(b, "┼"...)
		default:
			b = append(b, '+')
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
	lines  = strings.Repeat("─", 256)
)

// appendRepeat appends n copies of the single repeated character in block.
func appendRepeat(dst []byte, block string, n int) []byte {
	size := 1
	if block[0] >= utf8.RuneSelf {
		_, size = utf8.DecodeRuneInString(block)
	}
	for n > 0 {
		k := min(n, len(block)/size)
		dst = append(dst, block[:k*size]...)
		n -= k
	}
	return dst
}
