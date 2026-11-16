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
	for _, s := range []string{"0", "12", "-3.5", "+1", "1.234,56", "7%", "1e9", "2.5E-3", ".5"} {
		if !IsNumber(s) {
			t.Errorf("IsNumber(%q) = false", s)
		}
	}
	for _, s := range []string{"", "-", "abc", "1.", "2024-01-01", "12:30", "1e", "%", "1,", "5%x", "1x"} {
		if IsNumber(s) {
			t.Errorf("IsNumber(%q) = true", s)
		}
	}
}

func TestFit(t *testing.T) {
	tests := []struct {
		name   string
		widths []int
		limit  int
		want   []int
	}{
		{"fits", []int{3, 4}, 20, []int{3, 4}},
		{"shrinks widest", []int{3, 30, 10}, 30, []int{3, 11, 10}},
		{"keeps narrow", []int{2, 50}, 20, []int{2, 15}},
		{"drops columns", []int{20, 20, 20, 20}, 20, []int{9, 8}},
		{"single column", []int{100}, 10, []int{10}},
		{"empty", nil, 10, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
