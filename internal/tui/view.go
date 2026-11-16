package tui

import (
	"gridview/internal/layout"
)

const (
	reset     = "\x1b[0m"
	bold      = "\x1b[1m"
	dim       = "\x1b[2m"
	reverse   = "\x1b[7m"
	noReverse = "\x1b[27m"
	underline = "\x1b[4m"
	noUnder   = "\x1b[24m"
	clearEOL  = "\x1b[K"
)

// render appends one full frame to dst.
func (m *model) render(dst []byte) []byte {
	dst = append(dst, "\x1b[?2026h\x1b[H"...)
	cols, _ := m.visible(m.left)
	f := layout.Formatter{Cols: cols, Sep: sep, Cut: true}
	digits := m.gutter() - sepWidth

	dst = append(dst, dim...)
	dst = appendSpaces(dst, digits)
	dst = append(dst, sep...)
	dst = append(dst, reset+bold...)
	dst = f.AppendMarked(dst, from(m.header, m.left), m.col-m.left, underline, noUnder)
	dst = append(dst, reset+clearEOL+"\r\n"...)

	for i := range m.page() {
		r := m.top + i
		if r < m.rows {
			n := itoa(r + 1)
			dst = append(dst, dim...)
			dst = appendSpaces(dst, digits-len(n))
			dst = append(dst, n...)
			dst = append(dst, sep...)
			dst = append(dst, reset...)
			m.buf = m.store.Row(r, m.buf)
			if r == m.row {
				dst = append(dst, reverse...)
				dst = f.AppendMarked(dst, from(m.buf, m.left), m.col-m.left, noReverse, reverse)
				dst = append(dst, reset...)
			} else {
				dst = f.AppendRow(dst, from(m.buf, m.left))
			}
		}
		dst = append(dst, clearEOL+"\r\n"...)
	}

	dst = append(dst, reverse...)
	dst = appendFit(dst, m.status(), m.width)
	return append(dst, reset+"\x1b[?2026l"...)
}

func (m *model) status() string {
	if m.prompt {
		return "/" + string(m.input) + "▏"
	}
	s := " " + m.title + "   row " + itoa(min(m.row+1, m.rows)) + "/" + itoa(m.rows)
	if len(m.cols) > 0 {
		s += "   col " + itoa(m.col+1) + "/" + itoa(len(m.cols))
		if m.col < len(m.header) && m.header[m.col] != "" {
			s += " " + m.header[m.col]
		}
	}
	switch {
	case m.err != nil:
		s += "   error: " + m.err.Error()
	case m.loading:
		s += "   loading…"
	}
	if m.message != "" {
		return s + "   " + m.message
	}
	return s + "   ↑↓←→ PgUp PgDn g G  / search  n N  w wide  q quit"
}

func from(row []string, i int) []string {
	if i >= len(row) {
		return nil
	}
	return row[i:]
}

// appendFit appends s cut or padded to exactly w columns.
func appendFit(dst []byte, s string, w int) []byte {
	if layout.Width(s) > w {
		f := layout.Formatter{Cols: []layout.Column{{Width: w}}, Cut: true}
		return f.AppendRow(dst, []string{s})
	}
	dst = append(dst, s...)
	return appendSpaces(dst, w-layout.Width(s))
}

func appendSpaces(dst []byte, n int) []byte {
	for range n {
		dst = append(dst, ' ')
	}
	return dst
}
