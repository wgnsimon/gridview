package excel

import "testing"

func TestPickSheet(t *testing.T) {
	names := []string{"Daten", "2", "Leer"}
	tests := []struct {
		want  string
		index int
		ok    bool
	}{
		{"", 0, true},
		{"Daten", 0, true},
		{"leer", 2, true},
		{"2", 1, true},
		{"3", 2, true},
		{"4", 0, false},
		{"nope", 0, false},
	}
	for _, tt := range tests {
		i, err := PickSheet(names, tt.want)
		if (err == nil) != tt.ok || (tt.ok && i != tt.index) {
			t.Errorf("PickSheet(%q) = %d, %v", tt.want, i, err)
		}
	}
	if _, err := PickSheet(nil, ""); err == nil {
		t.Error("empty workbook: want error")
	}
}
