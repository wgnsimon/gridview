package table

import (
	"slices"
	"strings"
	"testing"
)

func TestSelect(t *testing.T) {
	header := []string{"id", "Name", "city", "2024"}
	data := [][]string{{"1", "Anna", "Berlin", "x"}, {"2"}}
	tests := []struct {
		spec   string
		header []string
		first  []string
		second []string
	}{
		{"name", []string{"Name"}, []string{"Anna"}, []string{""}},
		{"city,id", []string{"city", "id"}, []string{"Berlin", "1"}, []string{"", "2"}},
		{"2-3", []string{"Name", "city"}, []string{"Anna", "Berlin"}, []string{"", ""}},
		{"4, 1", []string{"2024", "id"}, []string{"x", "1"}, []string{"", "2"}},
		{"2024", []string{"2024"}, []string{"x"}, []string{""}},
		{"3-", []string{"city", "2024"}, []string{"Berlin", "x"}, []string{"", ""}},
	}
	for _, tt := range tests {
		r, err := Select(FromSlices(header, data), tt.spec)
		if err != nil {
			t.Errorf("%q: %v", tt.spec, err)
			continue
		}
		if !slices.Equal(r.Header(), tt.header) {
			t.Errorf("%q: header %q, want %q", tt.spec, r.Header(), tt.header)
		}
		rows := collect(t, r)
		if len(rows) != 2 || !slices.Equal(rows[0], tt.first) || !slices.Equal(rows[1], tt.second) {
			t.Errorf("%q: rows %q, want %q %q", tt.spec, rows, tt.first, tt.second)
		}
	}
}

func TestSelectErrors(t *testing.T) {
	header := []string{"id", "name"}
	for spec, want := range map[string]string{
		"town": `no column "town", columns are: id, name`,
		"3":    "no column 3, there are 2",
		"2-1":  `invalid range "2-1"`,
		"0":    "no column 0, there are 2",
		"id,":  "empty column name",
	} {
		_, err := Select(FromSlices(header, nil), spec)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want %q", spec, err, want)
		}
	}
}
