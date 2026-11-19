package tui

import "strings"

// startFilter opens the prompt to edit the filter.
func (m *model) startFilter() {
	if m.loading {
		m.message = "wait until loading has finished"
		return
	}
	m.prompt, m.filtering = true, true
	m.input = append(m.input[:0], []rune(m.filter)...)
}

// setFilter shows only the rows with a cell that contains q, ignoring case.
// An empty q shows all rows again.
func (m *model) setFilter(q string) {
	cur := m.current()
	m.filter = q
	m.applyOrder(cur)
	if m.rows == 0 {
		m.message = "no rows match: " + q
	}
}

// current returns the store row under the cursor, -1 if there is none.
func (m *model) current() int {
	if m.rows == 0 {
		return -1
	}
	return m.at(m.row)
}

// applyOrder rebuilds the rows on screen from the sort order and the filter.
// The cursor stays on store row cur if that is still shown.
func (m *model) applyOrder(cur int) {
	if m.sorted == nil && m.filter == "" {
		m.order, m.rows = nil, m.loaded
		if cur >= 0 {
			m.row = cur
		}
		return
	}
	q := strings.ToLower(m.filter)
	ascii := isASCII(q)
	m.order = m.order[:0]
	for i := range m.loaded {
		r := i
		if m.sorted != nil {
			r = int(m.sorted[i])
		}
		if q != "" && m.match(r, q, ascii) < 0 {
			continue
		}
		if r == cur {
			m.row = len(m.order)
		}
		m.order = append(m.order, int32(r))
	}
	if m.order == nil {
		m.order = []int32{}
	}
	m.rows = len(m.order)
}
