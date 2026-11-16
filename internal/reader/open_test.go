package reader

import (
	"os"
	"slices"
	"testing"

	"gridview/internal/reader/readertest"
	"gridview/internal/table"
)

// cell is an expected value at a data row and column.
type cell struct {
	row, col int
	want     string
}

// TestOpen reads every kind of fixture by path and from a stream.
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
		{file: "quoted.csv", header: []string{"id", "name", "note"}, rows: [][]string{{"1", "Schmidt, Peter", `says "hello"`}, {"2", "Lena", "two\nlines"}}},
		{file: "ragged.csv", header: []string{"a", "b", "c"}, rows: [][]string{{"1", "2"}, {"3", "4", "5", "6"}}},
		{file: "simple.csv", opt: Options{Delimiter: ';'}, header: []string{"id,name,city"}, count: 2},
		{file: "excel-local.csv", header: readertest.DataHeader, count: 5, cells: append(excelCSV, cell{0, 10, "Müller ? ??"})},
		{file: "excel-utf8.csv", header: readertest.DataHeader, count: 5, cells: append(excelCSV, cell{0, 10, "Müller ☕ 田中"})},
		{file: "excel-unicode.txt", header: readertest.DataHeader, count: 5, cells: append(excelCSV, cell{1, 10, "Emoji 😀"})},
		{file: "types.xlsx", header: readertest.DataHeader, rows: readertest.Data(readertest.Excel)},
		{file: "types.xls", header: readertest.DataHeader, rows: readertest.Data(readertest.Excel)},
	}
	for _, tt := range tests {
		t.Run(tt.file+"/"+string(tt.opt.Delimiter), func(t *testing.T) {
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
	for _, path := range []string{"testdata/missing.csv", broken} {
		if r, err := Open(path, Options{}); err == nil {
			r.Close()
			t.Errorf("Open(%q): want error", path)
		}
	}
}
