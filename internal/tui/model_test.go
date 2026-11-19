package tui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gridview/internal/layout"
	"gridview/internal/table"
)

func newTestModel(rows, cols, width, height int) *model {
	m := &model{title: "t.csv", store: &table.Store{}, width: width, height: height}
	var ms layout.Measure
	for c := range cols {
		m.header = append(m.header, fmt.Sprintf("col%d", c))
	}
	ms.AddHeader(m.header)
	for r := range rows {
		row := make([]string, cols)
		for c := range row {
			row[c] = fmt.Sprintf("r%dc%d", r, c)
		}
		m.store.Append(row)
		ms.Add(row)
	}
	m.rows, m.loaded = rows, rows
	m.cols = ms.Columns()
	return m
}

func press(m *model, keys ...Key) bool {
	for _, k := range keys {
		if m.handle(k) {
			return true
		}
	}
	return false
}

func runes(s string) []Key {
	var keys []Key
	for _, r := range s {
		keys = append(keys, Key{Rune: r})
	}
	return keys
}

func TestNavigation(t *testing.T) {
	m := newTestModel(100, 20, 40, 10)
	tests := []struct {
		keys          []Key
		row, top, col int
	}{
		{runes("jjj"), 3, 0, 0},
		{runes("k"), 2, 0, 0},
		{[]Key{{Code: KeyPgDn}}, 9, 7, 0},
		{runes("G"), 99, 93, 0},
		{[]Key{{Code: KeyDown}}, 99, 93, 0},
		{runes("g"), 99, 93, 0},
		{runes("g"), 0, 0, 0},
		{runes("$"), 0, 0, 19},
		{runes("h0"), 0, 0, 0},
	}
	for i, tt := range tests {
		if press(m, tt.keys...) {
			t.Fatalf("step %d quit", i)
		}
		if m.row != tt.row || m.top != tt.top || m.col != tt.col {
			t.Errorf("step %d: row %d top %d col %d, want %d %d %d", i, m.row, m.top, m.col, tt.row, tt.top, tt.col)
		}
	}
}

func TestViKeys(t *testing.T) {
	m := newTestModel(100, 20, 40, 10) // page of 7 rows
	ctrl := func(c KeyCode) []Key { return []Key{{Code: c}} }
	tests := []struct {
		name          string
		keys          []Key
		row, top, col int
	}{
		{"count j", runes("5j"), 5, 0, 0},
		{"count k", runes("3k"), 2, 0, 0},
		{"count l", runes("4l"), 2, 0, 4},
		{"count h", runes("10h"), 2, 0, 0},
		{"half page down", ctrl(KeyCtrlD), 5, 3, 0},
		{"half page up", ctrl(KeyCtrlU), 2, 0, 0},
		{"count G", runes("42G"), 41, 35, 0},
		{"count gg", runes("7gg"), 6, 6, 0},
		{"gg", runes("gg"), 0, 0, 0},
		{"L", runes("L"), 6, 0, 0},
		{"M", runes("M"), 3, 0, 0},
		{"H", runes("H"), 0, 0, 0},
		{"count L", runes("2L"), 5, 0, 0},
		{"count H", runes("3H"), 2, 0, 0},
		{"0 after count is a digit", runes("10j"), 12, 6, 0},
		{"0 alone is first column", runes("$0"), 12, 6, 0},
		{"esc drops count", append(runes("5"), append(ctrl(KeyEsc), runes("j")...)...), 13, 7, 0},
		{"esc drops g", append(runes("g"), append(ctrl(KeyEsc), runes("j")...)...), 14, 8, 0},
	}
	for _, tt := range tests {
		if press(m, tt.keys...) {
			t.Fatalf("%s: quit", tt.name)
		}
		if m.row != tt.row || m.top != tt.top || m.col != tt.col {
			t.Errorf("%s: row %d top %d col %d, want %d %d %d", tt.name, m.row, m.top, m.col, tt.row, tt.top, tt.col)
		}
	}
	if !press(m, Key{Code: KeyEsc}) {
		t.Error("esc without a pending count should quit")
	}
}

func TestBackwardSearch(t *testing.T) {
	m := newTestModel(50, 5, 60, 10)
	press(m, runes("20G")...)
	press(m, runes("?c3")...)
	if !strings.Contains(string(m.status(nil)), "?c3") {
		t.Errorf("prompt %q", string(m.status(nil)))
	}
	press(m, Key{Code: KeyEnter})
	if m.row != 18 || m.col != 3 {
		t.Fatalf("? found row %d col %d", m.row, m.col)
	}
	press(m, runes("n")...)
	if m.row != 17 {
		t.Errorf("n after ? went to row %d", m.row)
	}
	press(m, runes("N")...)
	if m.row != 18 {
		t.Errorf("N after ? went to row %d", m.row)
	}
	press(m, runes("3n")...)
	if m.row != 15 {
		t.Errorf("3n went to row %d", m.row)
	}
}

func TestHorizontalScrollKeepsCursorVisible(t *testing.T) {
	m := newTestModel(5, 20, 40, 10)
	for range 19 {
		press(m, Key{Code: KeyRight})
		idx, full := m.colsOnScreen()
		if pos := slices.Index(idx, m.col); pos < 0 || pos >= full {
			t.Fatalf("col %d not fully visible from left %d (%v)", m.col, m.left, idx)
		}
	}
}

func TestSearch(t *testing.T) {
	m := newTestModel(50, 5, 60, 10)
	press(m, runes("/R42C3")...)
	press(m, Key{Code: KeyEnter})
	if m.row != 42 || m.col != 3 || m.prompt {
		t.Fatalf("row %d col %d prompt %v", m.row, m.col, m.prompt)
	}
	press(m, runes("/1c4")...)
	press(m, Key{Code: KeyEnter})
	if m.row != 1 {
		t.Errorf("1c4 wrapped to row %d", m.row)
	}
	press(m, runes("N")...)
	if m.row != 41 {
		t.Errorf("N wrapped to row %d", m.row)
	}
	press(m, runes("/nothing")...)
	press(m, Key{Code: KeyEnter})
	if m.message != "not found: nothing" {
		t.Errorf("message %q", m.message)
	}
	press(m, runes("/x")...)
	if press(m, Key{Code: KeyEsc}) || m.prompt {
		t.Error("esc should close the prompt, not quit")
	}
}

func TestContainsFold(t *testing.T) {
	for _, tt := range []struct {
		s, q string
		want bool
	}{
		{"Hello World", "world", true},
		{"GRÖSSE", "grö", true},
		{"aAaAab", "aab", true},
		{"AAAA", "aab", false},
		{"ab", "abc", false},
		{"Mueller", "müller", false},
	} {
		if got := containsFold(tt.s, tt.q); got != tt.want {
			t.Errorf("containsFold(%q, %q) = %v", tt.s, tt.q, got)
		}
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestRender(t *testing.T) {
	tests := []struct {
		name          string
		rows, cols    int
		width, height int
		keys          string
		want          []string
	}{
		{"cursor row and column", 3, 3, 30, 7, "jl", []string{
			"  │ col0 │ col1 │ col2",
			"──┼──────┼──────┼─────",
			"1 │ r0c0 │ r0c1 │ r0c2",
			"2 │ r1c0 │ r1c1 │ r1c2",
			"3 │ r2c0 │ r2c1 │ r2c2",
			"",
			" t.csv   row 2/3   col 2/3 co…",
		}},
		{"scrolled right", 2, 6, 20, 5, "$", []string{
			"  │ col4 │ col5",
			"──┼──────┼─────",
			"1 │ r0c4 │ r0c5",
			"2 │ r1c4 │ r1c5",
			" t.csv   row 1/2   …",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(tt.rows, tt.cols, tt.width, tt.height)
			press(m, runes(tt.keys)...)
			got := strings.Split(ansi.ReplaceAllString(string(m.render(nil)), ""), "\r\n")
			if g, w := strings.Join(got, "\n"), strings.Join(tt.want, "\n"); g != w {
				t.Errorf("got:\n%s\nwant:\n%s", g, w)
			}
			for i, line := range got {
				if w := layout.Width(line); w > m.width {
					t.Errorf("line %d is %d wide", i, w)
				}
			}
		})
	}
}

func TestFreeze(t *testing.T) {
	m := newTestModel(3, 20, 40, 6)
	press(m, runes("lf$")...)
	if m.frozen != 2 {
		t.Fatalf("frozen %d, want 2", m.frozen)
	}
	got := screen(m)
	if want := "  │ col0 │ col1 │ col17 │ col18 │ col19"; got[0] != want {
		t.Errorf("header %q, want %q", got[0], want)
	}
	if want := "1 │ r0c0 │ r0c1 │ r0c17 │ r0c18 │ r0c19"; got[2] != want {
		t.Errorf("row %q, want %q", got[2], want)
	}
	press(m, runes("0")...)
	if got := screen(m); !strings.HasPrefix(got[0], "  │ col0 │ col1 │ col2 │") {
		t.Errorf("after 0 header %q", got[0])
	}
	press(m, runes("$hhhhhhhhhhhhhhhh")...)
	if m.col != 3 || m.left > 3 || m.left < 2 {
		t.Errorf("col %d left %d", m.col, m.left)
	}
	press(m, runes("f")...)
	if m.frozen != 0 {
		t.Errorf("second f should unfreeze, frozen %d", m.frozen)
	}
}

func TestFreezeKeepsCursorVisible(t *testing.T) {
	m := newTestModel(5, 20, 40, 10)
	press(m, runes("llf")...)
	for range 19 {
		press(m, Key{Code: KeyRight})
		idx, full := m.colsOnScreen()
		pos := slices.Index(idx, m.col)
		if pos < 0 || pos >= full {
			t.Fatalf("col %d not fully visible: %v (%d full)", m.col, idx, full)
		}
	}
}
