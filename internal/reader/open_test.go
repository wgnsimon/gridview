package reader

import (
	"os"
	"slices"
	"testing"

	"gridview/internal/reader/readertest"
	"gridview/internal/table"
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

// cell is an expected value at a data row and column.
type cell struct {
	row, col int
	want     string
}

func TestOpen(t *testing.T) {
	excelCSV := []cell{{0, 0, "Anna"}, {0, 2, "31.01.2024"}, {1, 0, `Bo, "quoted"`}, {2, 0, "two\nlines"}, {4, 10, "End"}}
	tests := []struct {
		file   string
		opt    Options
		header []string
		rows   [][]string
		count  int
		cells  []cell
	}{
		{file: "simple.csv", header: []string{"id", "name", "city"}, rows: [][]string{{"1", "Anna", "Berlin"}, {"2", "Ömer", "Köln"}}},
		{file: "quoted.csv", header: []string{"id", "name", "note"}, rows: [][]string{{"1", "Schmidt, Peter", `says "hello"`}, {"2", "Lena", "two\nlines"}}},
		{file: "ragged.csv", header: []string{"a", "b", "c"}, rows: [][]string{{"1", "2"}, {"3", "4", "5", "6"}}},
		{file: "simple.csv", opt: Options{Delimiter: ';'}, header: []string{"id,name,city"}, count: 2},
		{file: "excel-local.csv", header: readertest.DataHeader, count: 5, cells: append(excelCSV, cell{0, 10, "Müller ? ??"})},
		{file: "excel-utf8.csv", header: readertest.DataHeader, count: 5, cells: append(excelCSV, cell{0, 10, "Müller ☕ 田中"})},
		{file: "excel-unicode.txt", header: readertest.DataHeader, count: 5, cells: append(excelCSV, cell{1, 10, "Emoji 😀"})},
		{file: "types.xlsx", header: readertest.DataHeader, rows: readertest.Data(readertest.Excel)},
		{file: "types.xls", header: readertest.DataHeader, rows: readertest.Data(readertest.Excel)},
		{file: "libreoffice.xls", opt: Options{Sheet: "second sheet"}, header: []string{"x", "y"}, rows: [][]string{{"1", "2"}, {"2.5", "5"}}},
		{file: "types.xlsx", opt: Options{Sheet: "3"}},
		{file: "types.xlsx", opt: Options{Sheet: "Many"}, header: []string{"n", "text"}, count: 1200, cells: []cell{{1199, 1, "Row 1200 äöü ß"}}},
	}
	for _, tt := range tests {
		t.Run(tt.file+"/"+tt.opt.Sheet+string(tt.opt.Delimiter), func(t *testing.T) {
			check := func(t *testing.T, r table.Rows) {
				defer r.Close()
				if got := readertest.Trim(slices.Clone(r.Header())); !slices.Equal(got, readertest.Trim(tt.header)) {
					t.Errorf("header = %q, want %q", got, tt.header)
				}
				rows := readertest.ReadAll(t, r)
				if tt.rows != nil && !slices.EqualFunc(rows, tt.rows, slices.Equal) {
					t.Errorf("rows:\n got %q\nwant %q", rows, tt.rows)
				}
				if tt.count > 0 && len(rows) != tt.count {
					t.Errorf("got %d rows, want %d", len(rows), tt.count)
				}
				for _, c := range tt.cells {
					if c.row >= len(rows) || c.col >= len(rows[c.row]) || rows[c.row][c.col] != c.want {
						t.Errorf("cell %d,%d: want %q in %q", c.row, c.col, c.want, rows[min(c.row, len(rows)-1)])
					}
				}
			}
			path := "testdata/" + tt.file
			r, err := Open(path, tt.opt)
			if err != nil {
				t.Fatal(err)
			}
			check(t, r)

			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			r, err = OpenReader(f, tt.opt)
			if err != nil {
				t.Fatal(err)
			}
			check(t, r)
		})
	}
}

func TestOpenErrors(t *testing.T) {
	broken := t.TempDir() + "/broken.xlsx"
	if err := os.WriteFile(broken, []byte("PK\x03\x04 not really a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path string
		opt  Options
	}{
		{"testdata/missing.csv", Options{}},
		{"testdata/types.xlsx", Options{Sheet: "nope"}},
		{"testdata/types.xls", Options{Sheet: "9"}},
		{broken, Options{}},
	}
	for _, tt := range tests {
		if r, err := Open(tt.path, tt.opt); err == nil {
			r.Close()
			t.Errorf("Open(%q, %+v): want error", tt.path, tt.opt)
		}
	}
}
