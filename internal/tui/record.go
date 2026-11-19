package tui

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"gridview/internal/layout"
)

// recordLine is one screen line of the record view. label is empty on the
// lines a long value wraps onto.
type recordLine struct {
	label string
	text  string
	col   int
}

// handleRecord applies a key while the record view is open.
func (m *model) handleRecord(k Key) bool {
	switch {
	case k.Code == KeyCtrlC:
		return true
	case k.Code == KeyEsc || k.Code == KeyEnter || k.Rune == 'q':
		m.record = false
	case k.Code == KeyDown || k.Rune == 'j':
		m.recTop++
	case k.Code == KeyUp || k.Rune == 'k':
		m.recTop--
	case k.Code == KeyPgDn || k.Rune == ' ':
		m.recTop += m.recordPage()
	case k.Code == KeyPgUp || k.Rune == 'b':
		m.recTop -= m.recordPage()
	case k.Code == KeyHome || k.Rune == 'g':
		m.recTop = 0
	case k.Code == KeyEnd || k.Rune == 'G':
		m.recTop = len(m.recordLines())
	case k.Code == KeyRight || k.Rune == 'l':
		m.row++
		m.recTop = 0
	case k.Code == KeyLeft || k.Rune == 'h':
		m.row--
		m.recTop = 0
	}
	m.clamp()
	m.clampRecord()
	return false
}

// openRecord shows the current row with the current column in view.
func (m *model) openRecord() {
	if m.rows == 0 {
		return
	}
	m.record = true
	m.recTop = 0
	for i, l := range m.recordLines() {
		if l.col == m.col {
			if i >= m.recordPage() {
				m.recTop = i
			}
			break
		}
	}
	m.clampRecord()
}

func (m *model) recordPage() int {
	return max(1, m.height-1)
}

func (m *model) clampRecord() {
	m.recTop = max(0, min(m.recTop, len(m.recordLines())-m.recordPage()))
}

// labelWidth is the width of the column names, at most a third of the screen.
func (m *model) labelWidth() int {
	w := 1
	for i := range m.cols {
		w = max(w, layout.Width(m.label(i)))
	}
	return max(1, min(w, m.width/3))
}

func (m *model) label(col int) string {
	if col < len(m.header) && strings.TrimSpace(m.header[col]) != "" {
		return m.header[col]
	}
	return "#" + strconv.Itoa(col+1)
}

// recordLines lays out the current row as one name and value per column,
// with long values wrapped to the screen.
func (m *model) recordLines() []recordLine {
	if m.rows == 0 {
		return nil
	}
	m.buf = m.store.Row(m.row, m.buf)
	width := m.width - m.labelWidth() - sepWidth
	var out []recordLine
	for c := range m.cols {
		v := ""
		if c < len(m.buf) {
			v = m.buf[c]
		}
		for i, text := range wrap(v, width) {
			l := recordLine{text: text, col: c}
			if i == 0 {
				l.label = m.label(c)
			}
			out = append(out, l)
		}
	}
	return out
}

func (m *model) renderRecord(dst []byte) []byte {
	lines := m.recordLines()
	lw := m.labelWidth()
	for i := range m.recordPage() {
		if j := m.recTop + i; j < len(lines) {
			l := lines[j]
			if l.col == m.col && l.label != "" {
				dst = append(dst, reverse...)
			} else {
				dst = append(dst, bold...)
			}
			dst = appendFit(dst, l.label, lw)
			dst = append(dst, reset+dim+sep+reset...)
			dst = append(dst, l.text...)
		}
		dst = append(dst, clearEOL+"\r\n"...)
	}
	return dst
}

// wrap breaks s into lines of at most width columns, at spaces where
// possible. Line breaks in s are kept, other control characters become spaces.
func wrap(s string, width int) []string {
	width = max(width, 1)
	var out []string
	for para := range strings.SplitSeq(s, "\n") {
		out = wrapLine(out, strings.TrimSuffix(para, "\r"), width)
	}
	return out
}

func wrapLine(out []string, s string, width int) []string {
	var line []byte
	w, space := 0, -1
	for _, r := range s {
		if r < ' ' || r == 0x7F {
			r = ' '
		}
		rw := layout.RuneWidth(r)
		if w+rw > width && len(line) > 0 {
			switch {
			case r == ' ':
				out = append(out, string(line))
				line, w, space = line[:0], 0, -1
				continue
			case space > 0:
				out = append(out, string(line[:space]))
				line = append(line[:0], line[space+1:]...)
				w = layout.Width(string(line))
			default:
				out = append(out, string(line))
				line, w = line[:0], 0
			}
			space = -1
		}
		if r == ' ' {
			space = len(line)
		}
		line = utf8.AppendRune(line, r)
		w += rw
	}
	return append(out, string(line))
}
