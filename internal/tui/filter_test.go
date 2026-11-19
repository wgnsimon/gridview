package tui

import (
	"slices"
	"strings"
	"testing"
)

func filter(m *model, q string) {
	press(m, Key{Rune: '&'})
	for range 20 {
		press(m, Key{Code: KeyBackspace})
	}
	press(m, runes(q)...)
	press(m, Key{Code: KeyEnter})
}

func TestFilter(t *testing.T) {
	m := newSortModel()
	press(m, runes("j")...)
	filter(m, "O")
	if got, want := names(m), []string{"bo"}; !slices.Equal(got, want) {
		t.Fatalf("filter o: %q, want %q", got, want)
	}
	got := screen(m)
	if !strings.HasPrefix(got[2], "1 │ bo") || !strings.Contains(got[len(got)-1], `row 1/1`) || !strings.Contains(got[len(got)-1], `filter "O"`) {
		t.Errorf("screen:\n%s", strings.Join(got, "\n"))
	}
	filter(m, "n")
	press(m, runes("S")...)
	if got, want := names(m), []string{"Anna"}; !slices.Equal(got, want) {
		t.Errorf("filter n, sorted: %q, want %q", got, want)
	}
	filter(m, "")
	if got, want := names(m), []string{"cy", "bo", "Anna", ""}; !slices.Equal(got, want) {
		t.Errorf("filter cleared, still sorted: %q, want %q", got, want)
	}
	if m.row != 2 {
		t.Errorf("cursor should stay on Anna, now at %d", m.row)
	}
	filter(m, "nothing")
	if m.rows != 0 || !strings.Contains(m.message, "no rows match") {
		t.Errorf("rows %d message %q", m.rows, m.message)
	}
	screen(m)
	press(m, runes("jG/a")...)
	press(m, Key{Code: KeyEnter}, Key{Code: KeyEnter})
	if m.record {
		t.Error("record should not open without rows")
	}
}

func TestFilterPromptKeepsFilter(t *testing.T) {
	m := newSortModel()
	filter(m, "bo")
	press(m, Key{Rune: '&'})
	if string(m.input) != "bo" || !strings.Contains(screen(m)[7], "&bo") {
		t.Errorf("prompt should start with the filter, input %q", string(m.input))
	}
	press(m, Key{Code: KeyEsc})
	if m.filter != "bo" || m.prompt {
		t.Errorf("esc should keep the filter, filter %q", m.filter)
	}
}
