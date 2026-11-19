package tui

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"gridview/internal/layout"
	"gridview/internal/table"
)

func TestCompareCells(t *testing.T) {
	sorted := []string{"-3.5", "2", "10", "1.234,5", "1e4", "a2", "a10", "B", "file", "file2", "file10", "Zoo", ""}
	shuffled := slices.Clone(sorted)
	slices.Reverse(shuffled)
	keys := make([]sortKey, len(shuffled))
	for i, s := range shuffled {
		keys[i] = newSortKey(s)
	}
	slices.SortStableFunc(keys, compareKeys)
	var got []string
	for _, k := range keys {
		got = append(got, k.s)
	}
	if !slices.Equal(got, sorted) {
		t.Errorf("sorted to\n%q\nwant\n%q", got, sorted)
	}
}

func TestParseNumber(t *testing.T) {
	for s, want := range map[string]float64{
		"12": 12, "-3.5": -3.5, "3,5": 3.5, "1.234,56": 1234.56, "1,234.56": 1234.56,
		"1.234.567": 1234567, "25%": 25, "+7": 7, "1e3": 1000,
	} {
		if got, ok := parseNumber(s); !ok || got != want {
			t.Errorf("parseNumber(%q) = %v %v, want %v", s, got, ok, want)
		}
	}
	for _, s := range []string{"", "abc", "1-2", "12a"} {
		if _, ok := parseNumber(s); ok {
			t.Errorf("parseNumber(%q) should fail", s)
		}
	}
}

func newSortModel() *model {
	m := &model{title: "t.csv", store: &table.Store{}, width: 70, height: 8, header: []string{"name", "n"}}
	for _, r := range [][]string{{"bo", "10"}, {"Anna", "9"}, {"", "x"}, {"cy", ""}} {
		m.store.Append(r)
	}
	m.rows = 4
	m.cols = []layout.Column{{Width: 4}, {Width: 2}}
	return m
}

func names(m *model) []string {
	var out []string
	for r := range m.rows {
		out = append(out, m.store.Row(m.at(r), nil)[m.col])
	}
	return out
}

func TestSort(t *testing.T) {
	m := newSortModel()
	press(m, runes("s")...)
	if got, want := names(m), []string{"Anna", "bo", "cy", ""}; !slices.Equal(got, want) {
		t.Errorf("s: %q, want %q", got, want)
	}
	if m.row != 1 {
		t.Errorf("cursor should stay on bo, now at %d", m.row)
	}
	press(m, runes("S")...)
	if got, want := names(m), []string{"cy", "bo", "Anna", ""}; !slices.Equal(got, want) {
		t.Errorf("S: %q, want %q", got, want)
	}
	if got := screen(m); !strings.Contains(got[len(got)-1], "sorted by name ↓") {
		t.Errorf("status %q", got[len(got)-1])
	}
	if got := screen(m); !strings.HasPrefix(got[2], "4 │ cy") {
		t.Errorf("gutter should show the file row, got %q", got[2])
	}
	press(m, runes("S")...)
	if m.order != nil {
		t.Error("pressing S again should restore the file order")
	}
	press(m, runes("ls")...)
	if got, want := names(m), []string{"9", "10", "x", ""}; !slices.Equal(got, want) {
		t.Errorf("numbers: %q, want %q", got, want)
	}
}

func TestSortWhileLoading(t *testing.T) {
	m := newSortModel()
	m.loading = true
	press(m, runes("s")...)
	if m.order != nil || !strings.Contains(m.message, "loading") {
		t.Errorf("order %v message %q", m.order, m.message)
	}
}

func BenchmarkSort(b *testing.B) {
	m := &model{store: &table.Store{}, cols: make([]layout.Column, 2)}
	for i := range 1_000_000 {
		m.store.Append([]string{strconv.Itoa(i * 7919 % 1_000_003), "item" + strconv.Itoa(i*31%9973)})
	}
	m.rows = 1_000_000
	for b.Loop() {
		m.order = nil
		m.sortBy(0, false)
		m.order = nil
		m.sortBy(1, true)
	}
}
