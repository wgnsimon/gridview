package layout

// Measure collects column widths and detects numeric columns.
type Measure struct {
	cols []Column
	num  []bool
	text []bool
}

// AddHeader counts the header for widths but not for alignment.
func (m *Measure) AddHeader(row []string) {
	m.grow(len(row))
	for i, s := range row {
		m.cols[i].Width = max(m.cols[i].Width, Width(s))
	}
}

// Add counts a data row.
func (m *Measure) Add(row []string) {
	m.grow(len(row))
	for i, s := range row {
		m.cols[i].Width = max(m.cols[i].Width, Width(s))
		if s == "" || m.text[i] {
			continue
		}
		if IsNumber(s) {
			m.num[i] = true
		} else {
			m.text[i] = true
		}
	}
}

// Columns returns the measured columns; numeric ones are right aligned.
func (m *Measure) Columns() []Column {
	cols := make([]Column, len(m.cols))
	for i, c := range m.cols {
		cols[i] = Column{Width: c.Width, Right: m.num[i] && !m.text[i]}
	}
	return cols
}

func (m *Measure) grow(n int) {
	for len(m.cols) < n {
		m.cols = append(m.cols, Column{})
		m.num = append(m.num, false)
		m.text = append(m.text, false)
	}
}

// IsNumber reports whether s looks like a number: 12, -3.5, 1.234,56, 7%, 1e9.
func IsNumber(s string) bool {
	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	digits := 0
	for ; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			digits++
		case c == '.' || c == ',':
			if i+1 >= len(s) || s[i+1] < '0' || s[i+1] > '9' {
				return false
			}
		case c == '%':
			return digits > 0 && i == len(s)-1
		case c == 'e' || c == 'E':
			return digits > 0 && isExponent(s[i+1:])
		default:
			return false
		}
	}
	return digits > 0
}

func isExponent(s string) bool {
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}
