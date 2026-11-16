package layout

import (
	"slices"
	"testing"
)

func TestMeasure(t *testing.T) {
	var m Measure
	m.AddHeader([]string{"id", "name", "x"})
	m.Add([]string{"1", "Anna"})
	m.Add([]string{"123", "", "", "extra"})
	m.Add([]string{"-4,5", "Bo", "7%"})
	want := []Column{{4, true}, {4, false}, {2, true}, {5, false}}
	if got := m.Columns(); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestIsNumber(t *testing.T) {
	for _, s := range []string{"12", "-3.5", "1.234,56", "7%", "2.5E-3", ".5"} {
		if !IsNumber(s) {
			t.Errorf("IsNumber(%q) = false", s)
		}
	}
	for _, s := range []string{"", "-", "1.", "2024-01-01", "12:30", "1e", "%"} {
		if IsNumber(s) {
			t.Errorf("IsNumber(%q) = true", s)
		}
	}
}

func TestFit(t *testing.T) {
	tests := []struct {
		widths []int
		limit  int
		want   []int
	}{
		{[]int{3, 4}, 20, []int{3, 4}},
		{[]int{3, 30, 10}, 30, []int{3, 11, 10}},
		{[]int{20, 20, 20, 20}, 20, []int{9, 8}},
	}
	for _, tt := range tests {
		cols := make([]Column, len(tt.widths))
		for i, w := range tt.widths {
			cols[i].Width = w
		}
		n := Fit(cols, tt.limit, 3)
		got := make([]int, n)
		for i := range n {
			got[i] = cols[i].Width
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("Fit(%v, %d) = %v, want %v", tt.widths, tt.limit, got, tt.want)
		}
	}
}
