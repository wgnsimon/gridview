package xlsx

import (
	"io"
	"os"
	"slices"
	"strings"
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
	// LibreOffice saves without Excel's 1900 leap day, serial 60 is Feb 28 there.
	leap := map[string]string{"types.xlsx": "1900-02-29", "libreoffice.xlsx": "1900-02-28"}
	for _, file := range []string{"types.xlsx", "libreoffice.xlsx"} {
		t.Run(file, func(t *testing.T) {
			daten := slices.Clone(daten)
			daten[2] = slices.Clone(daten[2])
			daten[2][2] = leap[file]
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
			header, rows = readSheet(t, b, "viele")
			if len(rows) != 1200 || rows[0][1] != "Zeile 0001 äöü Größe €Ł" || rows[1199][0] != "1200" || rows[1199][1] != "Zeile 1200 äöü Größe" {
				t.Errorf("Viele: %d rows, first %q, last %q", len(rows), rows[0], rows[len(rows)-1])
			}
		})
	}
}

func TestColumn(t *testing.T) {
	tests := map[string]int{"A1": 0, "B2": 1, "Z9": 25, "AA1": 26, "az3": 51, "XFD1": 16383, "": -1, "12": -1, "ZZZZZ1": 16384}
	for ref, want := range tests {
		if got := column(ref); got != want {
			t.Errorf("column(%q) = %d, want %d", ref, got, want)
		}
	}
}

func TestNotAWorkbook(t *testing.T) {
	if _, err := Open(strings.NewReader("not a zip"), 9); err == nil {
		t.Error("want error")
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

// trim drops trailing empty cells; writers differ in whether they store them.
func trim(row []string) []string {
	for len(row) > 0 && row[len(row)-1] == "" {
		row = row[:len(row)-1]
	}
	if row == nil {
		return []string{}
	}
	return row
}
