package layout

import "testing"

func TestWidth(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"äöü", 3},
		{"Zoë Chen", 8},
		{"田中 太郎", 9},
		{"Kaffee ☕", 9},
		{"é", 1},
		{"a\nb", 3},
		{"\xff", 1},
	}
	for _, tt := range tests {
		if got := Width(tt.in); got != tt.want {
			t.Errorf("Width(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
