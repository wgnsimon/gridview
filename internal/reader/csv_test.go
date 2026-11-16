package reader

import (
	"io"
	"slices"
	"testing"

	"gridview/internal/table"
)

func TestOpenCSV(t *testing.T) {
	tests := []struct {
		file       string
		wantHeader []string
		wantRows   [][]string
	}{
		{
			file:       "testdata/simple.csv",
			wantHeader: []string{"id", "name", "city"},
			wantRows:   [][]string{{"1", "Anna", "Berlin"}, {"2", "Ömer", "Köln"}},
		},
		{
			file:       "testdata/quoted.csv",
			wantHeader: []string{"id", "name", "note"},
			wantRows:   [][]string{{"1", "Schmidt, Peter", `sagt "Hallo"`}, {"2", "Lena", "zwei\nZeilen"}},
		},
		{
			file:       "testdata/ragged.csv",
			wantHeader: []string{"a", "b", "c"},
			wantRows:   [][]string{{"1", "2"}, {"3", "4", "5", "6"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			rows, err := Open(tt.file, Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			if got := rows.Header(); !slices.Equal(got, tt.wantHeader) {
				t.Errorf("header = %q, want %q", got, tt.wantHeader)
			}
			if got := readAll(t, rows); !slices.EqualFunc(got, tt.wantRows, slices.Equal) {
				t.Errorf("rows = %q, want %q", got, tt.wantRows)
			}
		})
	}
}

func TestOpenMissingFile(t *testing.T) {
	if _, err := Open("testdata/missing.csv", Options{}); err == nil {
		t.Fatal("want error")
	}
}

func readAll(t *testing.T, rows table.Rows) [][]string {
	t.Helper()
	var all [][]string
	for {
		row, err := rows.Next()
		if err == io.EOF {
			return all
		}
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, slices.Clone(row))
	}
}
