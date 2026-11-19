package tui

import (
	"slices"
	"strings"
	"testing"

	"gridview/internal/layout"
	"gridview/internal/table"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		s     string
		width int
		want  []string
	}{
		{"", 10, []string{""}},
		{"short", 10, []string{"short"}},
		{"two words here", 9, []string{"two words", "here"}},
		{"abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
		{"line\nbreak", 20, []string{"line", "break"}},
		{"crlf\r\nend\tx", 20, []string{"crlf", "end x"}},
		{"日本語の文", 5, []string{"日本", "語の", "文"}},
	}
	for _, tt := range tests {
		if got := wrap(tt.s, tt.width); !slices.Equal(got, tt.want) {
			t.Errorf("wrap(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
		}
	}
}

func newRecordModel() *model {
	m := &model{title: "t.csv", store: &table.Store{}, width: 30, height: 8,
		header: []string{"id", "name", "note"}}
	m.store.Append([]string{"1", "Anna", "a long note that needs to wrap over lines"})
	m.store.Append([]string{"2", "Bo", "", "extra"})
	m.rows = 2
	m.cols = make([]layout.Column, 4)
	return m
}

func TestRecordView(t *testing.T) {
	m := newRecordModel()
	press(m, Key{Code: KeyEnter})
	if !m.record {
		t.Fatal("enter should open the record")
	}
	got := screen(m)
	want := []string{
		"id   │ 1",
		"name │ Anna",
		"note │ a long note that needs",
		"     │ to wrap over lines",
		"#4   │ ",
		"",
		"",
		" t.csv   row 1/2   record  ←→…",
	}
	if g, w := strings.Join(got, "\n"), strings.Join(want, "\n"); g != w {
		t.Errorf("got:\n%s\nwant:\n%s", g, w)
	}
	press(m, Key{Code: KeyRight})
	if m.row != 1 || !m.record {
		t.Fatalf("right should show the next row, row %d record %v", m.row, m.record)
	}
	if got := screen(m); got[3] != "#4   │ extra" {
		t.Errorf("unnamed column shown as %q", got[3])
	}
	if press(m, runes("q")...) || m.record {
		t.Error("q should close the record, not quit")
	}
}

func TestRecordScroll(t *testing.T) {
	m := newRecordModel()
	m.height = 3
	press(m, Key{Code: KeyEnter})
	press(m, runes("jjjjj")...)
	if m.recTop != 3 {
		t.Errorf("scrolled to %d, want the last page at 3", m.recTop)
	}
	press(m, runes("g")...)
	if m.recTop != 0 {
		t.Errorf("g scrolled to %d", m.recTop)
	}
}

func screen(m *model) []string {
	return strings.Split(ansi.ReplaceAllString(string(m.render(nil)), ""), "\r\n")
}
