package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

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

	record bool
	recTop int

	count   int  // pending count typed before a command, 0 if none
	g       bool // pending g of gg
	prompt  bool
	back    bool // search backwards, started with ?
	input   []rune
	query   string
	message string
	buf     []string
	fold    []byte
	line    []byte
	vis     []layout.Column
}

// page is the number of data rows on screen, below header and rule
// and above the status line.
func (m *model) page() int {
	return max(1, m.height-3)
}

// handle applies a key and reports whether to quit.
func (m *model) handle(k Key) bool {
	m.message = ""
	if m.prompt {
		m.edit(k)
		return false
	}
	if m.record {
		return m.handleRecord(k)
	}
	if k.Code == KeyRune && ('1' <= k.Rune && k.Rune <= '9' || k.Rune == '0' && m.count > 0) {
		m.count = min(m.count*10+int(k.Rune-'0'), maxCount)
		m.g = false
		return false
	}
	n, counted, g := max(1, m.count), m.count > 0, m.g
	m.count, m.g = 0, false
	half := max(1, m.page()/2)
	switch {
	case k.Code == KeyEsc && (counted || g):
		// Esc drops a pending count or g, like in vi.
	case k.Code == KeyCtrlC || k.Code == KeyEsc || k.Rune == 'q':
		return true
	case k.Code == KeyDown || k.Rune == 'j':
		m.row += n
	case k.Code == KeyUp || k.Rune == 'k':
		m.row -= n
	case k.Code == KeyPgDn || k.Rune == ' ':
		m.row += n * m.page()
		m.top += n * m.page()
	case k.Code == KeyPgUp || k.Rune == 'b':
		m.row -= n * m.page()
		m.top -= n * m.page()
	case k.Code == KeyCtrlD:
		m.row += n * half
		m.top += n * half
	case k.Code == KeyCtrlU:
		m.row -= n * half
		m.top -= n * half
	case k.Rune == 'g' && !g:
		if counted {
			m.count = n
		}
		m.g = true
		return false
	case k.Code == KeyHome || k.Rune == 'g' || k.Rune == 'G' && counted:
		m.row = n - 1
	case k.Code == KeyEnd || k.Rune == 'G':
		m.row = m.rows - 1
	case k.Rune == 'H':
		m.row = m.top + n - 1
	case k.Rune == 'M':
		m.row = m.top + (m.shown()-1)/2
	case k.Rune == 'L':
		m.row = max(m.top, m.top+m.shown()-n)
	case k.Code == KeyRight || k.Rune == 'l':
		m.col += n
	case k.Code == KeyLeft || k.Rune == 'h':
		m.col -= n
	case k.Rune == '0' || k.Rune == '^':
		m.col = 0
	case k.Rune == '$':
		m.col = len(m.cols) - 1
	case k.Code == KeyEnter:
		m.openRecord()
	case k.Rune == 'w':
		m.wide = !m.wide
	case k.Rune == '/' || k.Rune == '?':
		m.prompt = true
		m.back = k.Rune == '?'
		m.input = m.input[:0]
	case k.Rune == 'n':
		m.repeat(n, 1)
	case k.Rune == 'N':
		m.repeat(n, -1)
	}
	m.clamp()
	return false
}

// maxCount caps a typed count so arithmetic with it cannot overflow.
const maxCount = 1_000_000

// shown is the number of data rows on screen from top.
func (m *model) shown() int {
	return max(1, min(m.page(), m.rows-m.top))
}

// repeat jumps to the nth match, dir 1 in the search direction, -1 against it.
func (m *model) repeat(n, dir int) {
	if m.back {
		dir = -dir
	}
	for range min(n, max(1, m.rows)) {
		if !m.find(dir) {
			return
		}
	}
}

func (m *model) edit(k Key) {
	switch k.Code {
	case KeyEnter:
		m.prompt = false
		m.query = string(m.input)
		m.repeat(1, 1)
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

func (m *model) find(dir int) bool {
	if m.query == "" || m.rows == 0 {
		return false
	}
	q := strings.ToLower(m.query)
	ascii := isASCII(q)
	for step := 1; step <= m.rows; step++ {
		r := ((m.row+dir*step)%m.rows + m.rows) % m.rows
		// The stored bytes of a row rule out most rows without decoding them.
		if ascii && !containsASCII(m.store.Raw(r), q) {
			continue
		}
		m.buf = m.store.Row(r, m.buf)
		for c, s := range m.buf {
			hit := false
			if ascii {
				hit = containsASCII(s, q)
			} else {
				m.fold, hit = containsLower(m.fold, s, q)
			}
			if hit {
				m.row, m.col = r, c
				return true
			}
		}
	}
	m.message = "not found: " + m.query
	return false
}

// containsFold reports whether s contains the lower case q, ignoring case.
func containsFold(s, q string) bool {
	if isASCII(q) {
		return containsASCII(s, q)
	}
	_, hit := containsLower(nil, s, q)
	return hit
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// containsASCII is containsFold for an ASCII q. It jumps between the places
// where the first byte of q occurs in either case.
func containsASCII(s, q string) bool {
	n := len(q)
	if n == 0 {
		return true
	}
	limit := len(s) - n + 1
	if limit <= 0 {
		return false
	}
	lo, up := q[0], q[0]
	if lo >= 'a' && lo <= 'z' {
		up = lo - ('a' - 'A')
	}
	i, j := nextByte(s, lo, 0, limit), limit
	if up != lo {
		j = nextByte(s, up, 0, limit)
	}
	for {
		p := min(i, j)
		if p >= limit {
			return false
		}
		k := 1
		for k < n && lower(s[p+k]) == q[k] {
			k++
		}
		if k == n {
			return true
		}
		if p == i {
			i = nextByte(s, lo, p+1, limit)
		} else {
			j = nextByte(s, up, p+1, limit)
		}
	}
}

// nextByte returns the first index of c in s[from:limit], limit if there is none.
func nextByte(s string, c byte, from, limit int) int {
	if from >= limit {
		return limit
	}
	if k := strings.IndexByte(s[from:limit], c); k >= 0 {
		return from + k
	}
	return limit
}

// containsLower reports whether s in lower case contains q, which is not ASCII.
// buf is scratch space that is returned for reuse.
func containsLower(buf []byte, s, q string) ([]byte, bool) {
	if isASCII(s) {
		return buf, false
	}
	buf = buf[:0]
	for _, r := range s {
		buf = utf8.AppendRune(buf, unicode.ToLower(r))
	}
	return buf, strings.Contains(unsafe.String(unsafe.SliceData(buf), len(buf)), q)
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
