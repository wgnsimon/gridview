package tui

import (
	"strings"
	"unicode/utf8"

	"gridview/internal/layout"
	"gridview/internal/table"
)

const (
	maxCell  = 40
	sep      = " │ "
	sepWidth = 3
)

// model is the view state. It knows nothing about terminals.
type model struct {
	title   string
	header  []string
	store   *table.Store
	cols    []layout.Column
	rows    int
	loading bool
	err     error

	width, height int
	top, row      int
	left, col     int
	wide          bool

	prompt  bool
	input   []rune
	query   string
	message string
	buf     []string
	line    []byte
	vis     []layout.Column
}

func (m *model) page() int {
	return max(1, m.height-2)
}

// handle applies a key and reports whether to quit.
func (m *model) handle(k Key) bool {
	m.message = ""
	if m.prompt {
		m.edit(k)
		return false
	}
	switch {
	case k.Code == KeyCtrlC || k.Code == KeyEsc || k.Rune == 'q':
		return true
	case k.Code == KeyDown || k.Rune == 'j':
		m.row++
	case k.Code == KeyUp || k.Rune == 'k':
		m.row--
	case k.Code == KeyPgDn || k.Rune == ' ':
		m.row += m.page()
		m.top += m.page()
	case k.Code == KeyPgUp || k.Rune == 'b':
		m.row -= m.page()
		m.top -= m.page()
	case k.Code == KeyHome || k.Rune == 'g':
		m.row = 0
	case k.Code == KeyEnd || k.Rune == 'G':
		m.row = m.rows - 1
	case k.Code == KeyRight || k.Rune == 'l':
		m.col++
	case k.Code == KeyLeft || k.Rune == 'h':
		m.col--
	case k.Rune == '0' || k.Rune == '^':
		m.col = 0
	case k.Rune == '$':
		m.col = len(m.cols) - 1
	case k.Rune == 'w':
		m.wide = !m.wide
	case k.Rune == '/':
		m.prompt = true
		m.input = m.input[:0]
	case k.Rune == 'n':
		m.find(1)
	case k.Rune == 'N':
		m.find(-1)
	}
	m.clamp()
	return false
}

func (m *model) edit(k Key) {
	switch k.Code {
	case KeyEnter:
		m.prompt = false
		m.query = string(m.input)
		m.find(1)
		m.clamp()
	case KeyEsc, KeyCtrlC:
		m.prompt = false
	case KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	case KeyRune:
		m.input = append(m.input, k.Rune)
	}
}

func (m *model) find(dir int) {
	if m.query == "" || m.rows == 0 {
		return
	}
	q := strings.ToLower(m.query)
	for step := 1; step <= m.rows; step++ {
		r := ((m.row+dir*step)%m.rows + m.rows) % m.rows
		m.buf = m.store.Row(r, m.buf)
		for c, s := range m.buf {
			if containsFold(s, q) {
				m.row, m.col = r, c
				return
			}
		}
	}
	m.message = "not found: " + m.query
}

// containsFold reports whether s contains the lower case q, ignoring case.
func containsFold(s, q string) bool {
	for i := 0; i < len(q); i++ {
		if q[i] >= utf8.RuneSelf {
			return strings.Contains(strings.ToLower(s), q)
		}
	}
	for i := 0; i+len(q) <= len(s); i++ {
		j := 0
		for j < len(q) && lower(s[i+j]) == q[j] {
			j++
		}
		if j == len(q) {
			return true
		}
	}
	return false
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

func (m *model) clamp() {
	m.row = max(0, min(m.row, m.rows-1))
	m.top = min(m.top, max(0, m.rows-m.page()))
	if m.row < m.top {
		m.top = m.row
	}
	if m.row >= m.top+m.page() {
		m.top = m.row - m.page() + 1
	}
	m.top = max(0, m.top)
	m.col = max(0, min(m.col, len(m.cols)-1))
	m.left = max(0, min(m.left, m.col))
	for m.left < m.col {
		if _, full := m.visible(m.left); m.col < m.left+full {
			break
		}
		m.left++
	}
}

// visible returns the columns shown from left; the last may be cut.
// full counts the columns that fit completely.
func (m *model) visible(left int) ([]layout.Column, int) {
	avail := m.width - m.gutter()
	out := m.vis[:0]
	defer func() { m.vis = out[:0] }()
	used := 0
	for i := left; i < len(m.cols); i++ {
		c := m.cols[i]
		if !m.wide {
			c.Width = min(c.Width, maxCell)
		}
		c.Width = max(c.Width, 1)
		gap := 0
		if len(out) > 0 {
			gap = sepWidth
		}
		if used+gap+c.Width <= avail {
			out = append(out, c)
			used += gap + c.Width
			continue
		}
		if rest := avail - used - gap; rest > 0 {
			c.Width = rest
			return append(out, c), len(out)
		}
		break
	}
	return out, len(out)
}

func (m *model) gutter() int {
	return digits(max(m.rows, 1)) + sepWidth
}

func digits(n int) int {
	d := 1
	for n >= 10 {
		n /= 10
		d++
	}
	return d
}
