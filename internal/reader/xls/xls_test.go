package xls

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"testing"

	"gridview/internal/table"
)

var daten = [][]string{
	{"Anna", "42", "2024-01-31", "13:45:00", "2024-01-31 13:45:00", "25%", "TRUE", "84", "#DIV/0!", "", "Müller ☕ 田中"},
	{`Bo, "quoted"`, "-3.5", "1900-03-01", "", "", "12.50%", "FALSE", "0.3", "#N/A", "", "Emoji 😀"},
	{"zwei\nZeilen", "1234567.89", "1900-02-29", "", "", "", "", `Anna-Bo, "quoted"`},
	{},
	{"00123", "", "", "", "", "", "", "", "", "", "Ende"},
}

func TestFixtures(t *testing.T) {
	// LibreOffice stores 0 instead of the cached text of a string formula.
	formula := map[string]string{"types.xls": `Anna-Bo, "quoted"`, "libreoffice.xls": "0"}
	for _, file := range []string{"types.xls", "libreoffice.xls"} {
		t.Run(file, func(t *testing.T) {
			daten := slices.Clone(daten)
			daten[2] = slices.Clone(daten[2])
			daten[2][7] = formula[file]
			b := openBook(t, "../testdata/"+file)
			if got, want := b.Sheets(), []string{"Daten", "Zweites Blatt", "Leer", "Viele"}; !slices.Equal(got, want) {
				t.Fatalf("sheets = %q, want %q", got, want)
			}
			header, rows := readSheet(t, b, "")
			if want := []string{"Text", "Zahl", "Datum", "Zeit", "DatumZeit", "Prozent", "Bool", "Formel", "Fehler", "Leer", "Unicode"}; !slices.Equal(trim(header), want) {
				t.Errorf("header = %q", header)
			}
			if !slices.EqualFunc(rows, daten, slices.Equal) {
				t.Errorf("Daten rows:\n got %q\nwant %q", rows, daten)
			}
			header, rows = readSheet(t, b, "2")
			if !slices.Equal(trim(header), []string{"x", "y"}) || !slices.EqualFunc(rows, [][]string{{"1", "2"}, {"2.5", "5"}}, slices.Equal) {
				t.Errorf("Zweites Blatt = %q %q", header, rows)
			}
			header, rows = readSheet(t, b, "Leer")
			if len(header) != 0 || len(rows) != 0 {
				t.Errorf("Leer = %q %q", header, rows)
			}
			_, rows = readSheet(t, b, "Viele")
			if len(rows) != 1200 {
				t.Fatalf("Viele: %d rows", len(rows))
			}
			for i, row := range rows {
				want := fmtRow(i + 1)
				if !slices.Equal(row, want) {
					t.Fatalf("Viele row %d = %q, want %q", i+1, row, want)
				}
			}
		})
	}
}

func fmtRow(i int) []string {
	s := fmt.Sprintf("Zeile %04d äöü Größe", i)
	if i%2 == 1 {
		s += " €Ł"
	}
	return []string{strconv.Itoa(i), s}
}

func TestRK(t *testing.T) {
	tests := []struct {
		v    uint32
		want float64
	}{
		{42<<2 | 2, 42},
		{0xFFFFFFE6, -7},
		{1234<<2 | 3, 12.34},
		{0x3FF00000, 1},
		{0x3FF00000 | 1, 0.01},
	}
	for _, tt := range tests {
		if got := rk(tt.v); got != tt.want {
			t.Errorf("rk(%#x) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestStrReaderAcrossContinue(t *testing.T) {
	// "abcdef": cch 6, compressed, split after "ab"; the second part switches to UTF-16.
	segs := [][]byte{
		{6, 0, 0, 'a', 'b'},
		{1, 'c', 0, 'd', 0, 'e', 0, 'f', 0},
	}
	s := &strReader{segs: segs}
	if got, ok := s.str(false); !ok || got != "abcdef" {
		t.Errorf("got %q, %v", got, ok)
	}
	rich := [][]byte{{2, 0, 8, 1, 0, 'h', 'i', 1, 2}, {3, 4}, {1, 0, 0, 'x'}}
	s = &strReader{segs: rich}
	if got, ok := s.str(false); !ok || got != "hi" {
		t.Errorf("rich: got %q, %v", got, ok)
	}
	if got, ok := s.str(false); !ok || got != "x" {
		t.Errorf("after rich: got %q, %v", got, ok)
	}
	if _, ok := (&strReader{segs: [][]byte{{5, 0, 0, 'a'}}}).str(false); ok {
		t.Error("truncated string: want failure")
	}
}

func openBook(t *testing.T, name string) *Book {
	t.Helper()
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(f, st.Size())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readSheet(t *testing.T, b *Book, sheet string) ([]string, [][]string) {
	t.Helper()
	r, err := b.Rows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	return r.Header(), readAll(t, r)
}

func readAll(t *testing.T, r table.Rows) [][]string {
	t.Helper()
	var all [][]string
	for {
		row, err := r.Next()
		if err == io.EOF {
			return all
		}
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, trim(slices.Clone(row)))
	}
}

func trim(row []string) []string {
	for len(row) > 0 && row[len(row)-1] == "" {
		row = row[:len(row)-1]
	}
	if row == nil {
		return []string{}
	}
	return row
}
