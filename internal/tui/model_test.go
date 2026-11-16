package tui

import (
	"fmt"
	"regexp"
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
	m.rows = rows
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
		{[]Key{{Code: KeyPgDn}}, 10, 8, 0},
		{runes("G"), 99, 92, 0},
		{[]Key{{Code: KeyDown}}, 99, 92, 0},
		{runes("g"), 0, 0, 0},
		{[]Key{{Code: KeyUp}}, 0, 0, 0},
		{runes("lll"), 0, 0, 3},
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

func TestHorizontalScrollKeepsCursorVisible(t *testing.T) {
	m := newTestModel(5, 20, 40, 10)
	for range 19 {
		press(m, Key{Code: KeyRight})
		cols, full := m.visible(m.left)
		if m.col < m.left || m.col >= m.left+full {
			t.Fatalf("col %d not fully visible from left %d (%d cols)", m.col, m.left, len(cols))
		}
	}
	press(m, runes("0")...)
	if m.left != 0 {
		t.Errorf("left = %d after 0", m.left)
	}
}

func TestQuit(t *testing.T) {
	for _, k := range []Key{{Rune: 'q'}, {Code: KeyEsc}, {Code: KeyCtrlC}} {
		if !press(newTestModel(1, 1, 20, 5), k) {
			t.Errorf("%v did not quit", k)
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
	press(m, runes("n")...)
	if m.row != 42 {
		t.Errorf("n wrapped to row %d", m.row)
	}
	press(m, runes("/r1c")...)
	press(m, Key{Code: KeyBackspace}, Key{Rune: 'c'}, Key{Code: KeyEnter})
	if m.row != 1 {
		t.Errorf("wrapped search row %d", m.row)
	}
	press(m, runes("/1c4")...)
	press(m, Key{Code: KeyEnter})
	if m.row != 11 {
		t.Errorf("1c4 row %d", m.row)
	}
	press(m, runes("NN")...)
	if m.row != 41 {
		t.Errorf("NN wrapped to row %d", m.row)
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
		{"Hello", "hello!", false},
		{"Größe", "grö", true},
		{"GRÖSSE", "grö", true},
		{"abc", "", true},
	} {
		if got := containsFold(tt.s, tt.q); got != tt.want {
			t.Errorf("containsFold(%q, %q) = %v", tt.s, tt.q, got)
		}
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func screen(m *model) []string {
	return strings.Split(ansi.ReplaceAllString(string(m.render(nil)), ""), "\r\n")
}

func TestRender(t *testing.T) {
	tests := []struct {
		name          string
		rows, cols    int
		width, height int
		keys          string
		want          []string
	}{
		{"cursor row and column", 3, 3, 30, 6, "jl", []string{
			"  │ col0 │ col1 │ col2",
			"1 │ r0c0 │ r0c1 │ r0c2",
			"2 │ r1c0 │ r1c1 │ r1c2",
			"3 │ r2c0 │ r2c1 │ r2c2",
			"",
			" t.csv   row 2/3   col 2/3 co…",
		}},
		{"cuts last column", 1, 3, 20, 4, "", []string{
			"  │ col0 │ col1 │ c…",
			"1 │ r0c0 │ r0c1 │ r…",
			"",
			" t.csv   row 1/1   …",
		}},
		{"scrolled right", 2, 6, 20, 4, "$", []string{
			"  │ col4 │ col5",
			"1 │ r0c4 │ r0c5",
			"2 │ r1c4 │ r1c5",
			" t.csv   row 1/2   …",
		}},
		{"empty", 0, 0, 20, 4, "", []string{
			"  │ ",
			"",
			"",
			" t.csv   row 0/0   …",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(tt.rows, tt.cols, tt.width, tt.height)
			press(m, runes(tt.keys)...)
			got := screen(m)
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
