package reader

import (
	"os"
	"slices"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		head []byte
		want format
	}{
		{[]byte("PK\x03\x04rest"), formatXLSX},
		{[]byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1"), formatXLS},
		{[]byte("a,b"), formatCSV},
		{[]byte("PK"), formatCSV},
		{nil, formatCSV},
	}
	for _, tt := range tests {
		if got := detect(tt.head); got != tt.want {
			t.Errorf("detect(%q) = %d, want %d", tt.head, got, tt.want)
		}
	}
}

func TestOpenFormats(t *testing.T) {
	tests := []struct {
		file  string
		sheet string
		first []string
	}{
		{"testdata/types.xlsx", "", []string{"Anna", "42", "2024-01-31"}},
		{"testdata/types.xlsx", "Zweites Blatt", []string{"1", "2"}},
		{"testdata/types.xls", "", []string{"Anna", "42", "2024-01-31"}},
		{"testdata/libreoffice.xls", "viele", []string{"1", "Zeile 0001 äöü Größe €Ł"}},
		{"testdata/excel-local.csv", "", []string{"Anna", "42", "31.01.2024"}},
		{"testdata/excel-utf8.csv", "", []string{"Anna", "42", "31.01.2024"}},
		{"testdata/excel-unicode.txt", "", []string{"Anna", "42", "31.01.2024"}},
	}
	for _, tt := range tests {
		t.Run(tt.file+tt.sheet, func(t *testing.T) {
			check := func(t *testing.T, rows [][]string) {
				if len(rows) == 0 || !slices.Equal(rows[0][:len(tt.first)], tt.first) {
					t.Errorf("first row = %q, want prefix %q", rows[0], tt.first)
				}
			}
			r, err := Open(tt.file, Options{Sheet: tt.sheet})
			if err != nil {
				t.Fatal(err)
			}
			check(t, readAll(t, r))
			r.Close()

			f, err := os.Open(tt.file)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			r, err = OpenReader(f, Options{Sheet: tt.sheet})
			if err != nil {
				t.Fatal(err)
			}
			check(t, readAll(t, r))
		})
	}
}

func TestExcelCSVExports(t *testing.T) {
	for _, file := range []string{"testdata/excel-local.csv", "testdata/excel-utf8.csv", "testdata/excel-unicode.txt"} {
		r, err := Open(file, Options{})
		if err != nil {
			t.Fatal(err)
		}
		rows := readAll(t, r)
		r.Close()
		if len(rows) != 5 || rows[2][0] != "zwei\nZeilen" || rows[4][10] != "Ende" {
			t.Errorf("%s: got %q", file, rows)
		}
		if want := "Müller"; file != "testdata/excel-local.csv" && rows[0][10][:len(want)] != want {
			t.Errorf("%s: unicode cell = %q", file, rows[0][10])
		}
	}
}
