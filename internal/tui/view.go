package tui

import (
	"slices"
	"strconv"
	"unicode/utf8"
	"unsafe"

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
	if m.record {
		dst = m.renderRecord(dst)
		return m.appendStatus(dst)
	}
	m.layoutCols(m.left)
	f := layout.Formatter{Cols: m.vis, Sep: sep, Cut: true}
	mark := slices.Index(m.idx, m.col)
	width := m.gutter() - sepWidth

	dst = append(dst, dim...)
	dst = appendSpaces(dst, width)
	dst = append(dst, sep...)
	dst = append(dst, reset+bold...)
	dst = f.AppendMarked(dst, m.pick(m.header), mark, underline, noUnder)
	dst = append(dst, reset+clearEOL+"\r\n"...)

	dst = append(dst, dim...)
	dst = appendLines(dst, width)
	dst = append(dst, "─┼─"...)
	dst = f.AppendRule(dst)
	dst = append(dst, reset+clearEOL+"\r\n"...)

	for i := range m.page() {
		r := m.top + i
		if r < m.rows {
			dst = append(dst, dim...)
			dst = appendSpaces(dst, width-digits(r+1))
			dst = strconv.AppendInt(dst, int64(r+1), 10)
			dst = append(dst, sep...)
			dst = append(dst, reset...)
			m.buf = m.store.Row(r, m.buf)
			if r == m.row {
				dst = append(dst, reverse...)
				dst = f.AppendMarked(dst, m.pick(m.buf), mark, noReverse, reverse)
				dst = append(dst, reset...)
			} else {
				dst = f.AppendRow(dst, m.pick(m.buf))
			}
		}
		dst = append(dst, clearEOL+"\r\n"...)
	}
	return m.appendStatus(dst)
}

func (m *model) appendStatus(dst []byte) []byte {
	dst = append(dst, reverse...)
	m.line = m.status(m.line[:0])
	dst = appendFit(dst, unsafe.String(unsafe.SliceData(m.line), len(m.line)), m.width)
	return append(dst, reset+"\x1b[?2026l"...)
}

func (m *model) status(b []byte) []byte {
	if m.prompt {
		if m.back {
			b = append(b, '?')
		} else {
			b = append(b, '/')
		}
		for _, r := range m.input {
			b = utf8.AppendRune(b, r)
		}
		return append(b, "▏"...)
	}
	b = append(append(append(b, ' '), m.title...), "   row "...)
	b = strconv.AppendInt(b, int64(min(m.row+1, m.rows)), 10)
	b = strconv.AppendInt(append(b, '/'), int64(m.rows), 10)
	if len(m.cols) > 0 && !m.record {
		b = strconv.AppendInt(append(b, "   col "...), int64(m.col+1), 10)
		b = strconv.AppendInt(append(b, '/'), int64(len(m.cols)), 10)
		if m.col < len(m.header) && m.header[m.col] != "" {
			b = append(append(b, ' '), m.header[m.col]...)
		}
	}
	if m.frozen > 0 && !m.record {
		b = strconv.AppendInt(append(b, "   frozen "...), int64(m.frozen), 10)
	}
	switch {
	case m.err != nil:
		b = append(append(b, "   error: "...), m.err.Error()...)
	case m.loading:
		b = append(b, "   loading…"...)
	}
	if m.message != "" {
		return append(append(b, "   "...), m.message...)
	}
	if m.record {
		return append(b, "   record  ←→ row  ↑↓ scroll  Esc back"...)
	}
	if m.count > 0 || m.g {
		b = append(b, "   "...)
		if m.count > 0 {
			b = strconv.AppendInt(b, int64(m.count), 10)
		}
		if m.g {
			b = append(b, 'g')
		}
		return b
	}
	return append(b, "   hjkl ^D ^U gg G  / ? search  n N  Enter record  f freeze  w wide  q quit"...)
}

// pick returns the cells of row in the columns on screen.
func (m *model) pick(row []string) []string {
	m.cells = m.cells[:0]
	for _, i := range m.idx {
		c := ""
		if i < len(row) {
			c = row[i]
		}
		m.cells = append(m.cells, c)
	}
	return m.cells
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

func appendLines(dst []byte, n int) []byte {
	for range n {
		dst = append(dst, "─"...)
	}
	return dst
}

func appendSpaces(dst []byte, n int) []byte {
	for range n {
		dst = append(dst, ' ')
	}
	return dst
}
