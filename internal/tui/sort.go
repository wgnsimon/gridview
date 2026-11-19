package tui

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"gridview/internal/layout"
)

// at returns the store index of row r on screen.
func (m *model) at(r int) int {
	if m.order == nil {
		return r
	}
	return int(m.order[r])
}

// sortBy orders the rows by column col. Sorting again by the same column
// and direction restores the file order. The cursor stays on its row.
func (m *model) sortBy(col int, desc bool) {
	if m.loading {
		m.message = "wait until loading has finished"
		return
	}
	if m.rows == 0 || col >= len(m.cols) {
		return
	}
	cur := m.at(m.row)
	if m.order != nil && m.sortCol == col && m.sortDesc == desc {
		m.order = nil
		m.row = cur
		return
	}
	keys := make([]sortKey, m.rows)
	for r := range keys {
		m.buf = m.store.Row(r, m.buf)
		cell := ""
		if col < len(m.buf) {
			cell = m.buf[col]
		}
		keys[r] = newSortKey(cell)
		keys[r].row = int32(r)
	}
	slices.SortStableFunc(keys, func(a, b sortKey) int {
		c := compareKeys(a, b)
		if desc && a.kind != kindEmpty && b.kind != kindEmpty {
			c = -c
		}
		return c
	})
	if m.order == nil {
		m.order = make([]int32, m.rows)
	}
	for i, k := range keys {
		m.order[i] = k.row
		if int(k.row) == cur {
			m.row = i
		}
	}
	m.sortCol, m.sortDesc = col, desc
}

const (
	kindNumber uint8 = iota
	kindText
	kindEmpty
)

// sortKey orders numbers by value before text in natural order;
// empty cells come last.
type sortKey struct {
	kind uint8
	num  float64
	s    string
	row  int32
}

func newSortKey(s string) sortKey {
	t := strings.TrimSpace(s)
	if t == "" {
		return sortKey{kind: kindEmpty, s: s}
	}
	if f, ok := parseNumber(t); ok {
		return sortKey{kind: kindNumber, num: f, s: s}
	}
	return sortKey{kind: kindText, s: s}
}

func compareKeys(a, b sortKey) int {
	if c := cmp.Compare(a.kind, b.kind); c != 0 {
		return c
	}
	switch a.kind {
	case kindNumber:
		return cmp.Compare(a.num, b.num)
	case kindText:
		return compareNatural(a.s, b.s)
	}
	return 0
}

// parseNumber reads the numbers IsNumber accepts. With both . and , the
// last one is the decimal separator; a single , is a decimal comma.
func parseNumber(s string) (float64, bool) {
	if !layout.IsNumber(s) {
		return 0, false
	}
	s = strings.TrimSuffix(s, "%")
	dots, commas := strings.Count(s, "."), strings.Count(s, ",")
	switch {
	case dots > 0 && commas > 0:
		if strings.LastIndexByte(s, '.') > strings.LastIndexByte(s, ',') {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", ".")
		}
	case commas > 1:
		s = strings.ReplaceAll(s, ",", "")
	case commas == 1:
		s = strings.Replace(s, ",", ".", 1)
	case dots > 1:
		s = strings.ReplaceAll(s, ".", "")
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// compareNatural compares ignoring case, with runs of digits compared by
// value, so file2 comes before file10.
func compareNatural(a, b string) int {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			ei, ej := digitsEnd(a, i), digitsEnd(b, j)
			na, nb := strings.TrimLeft(a[i:ei], "0"), strings.TrimLeft(b[j:ej], "0")
			if c := cmp.Compare(len(na), len(nb)); c != 0 {
				return c
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
			i, j = ei, ej
			continue
		}
		ra, sa := utf8.DecodeRuneInString(a[i:])
		rb, sb := utf8.DecodeRuneInString(b[j:])
		if c := cmp.Compare(unicode.ToLower(ra), unicode.ToLower(rb)); c != 0 {
			return c
		}
		i, j = i+sa, j+sb
	}
	if c := cmp.Compare(len(a)-i, len(b)-j); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func digitsEnd(s string, i int) int {
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	return i
}
