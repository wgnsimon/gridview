// Package readertest checks readers against the shared fixtures in reader/testdata.
package readertest

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"testing"

	"gridview/internal/table"
)

// Book is what xlsx and xls workbooks offer.
type Book interface {
	Sheets() []string
	Rows(sheet string) (table.Rows, error)
}

// Quirks are the places where writers legitimately store different values.
type Quirks struct {
	LeapDay string // serial 60: Excel shows 1900-02-29, LibreOffice 1900-02-28
	Formula string // cached text of a string formula, LibreOffice stores 0
}

var Excel = Quirks{LeapDay: "1900-02-29", Formula: `Anna-Bo, "quoted"`}

var DatenHeader = []string{"Text", "Zahl", "Datum", "Zeit", "DatumZeit", "Prozent", "Bool", "Formel", "Fehler", "Leer", "Unicode"}

// Daten returns the rows of the first fixture sheet.
func Daten(q Quirks) [][]string {
	return [][]string{
		{"Anna", "42", "2024-01-31", "13:45:00", "2024-01-31 13:45:00", "25%", "TRUE", "84", "#DIV/0!", "", "Müller ☕ 田中"},
		{`Bo, "quoted"`, "-3.5", "1900-03-01", "", "", "12.50%", "FALSE", "0.3", "#N/A", "", "Emoji 😀"},
		{"zwei\nZeilen", "1234567.89", q.LeapDay, "", "", "", "", q.Formula},
		{},
		{"00123", "", "", "", "", "", "", "", "", "", "Ende"},
	}
}

// Viele returns row i (1-based) of the sheet with 1200 shared strings.
func Viele(i int) []string {
	s := fmt.Sprintf("Zeile %04d äöü Größe", i)
	if i%2 == 1 {
		s += " €Ł"
	}
	return []string{strconv.Itoa(i), s}
}

// CheckWorkbook compares all four fixture sheets of b.
func CheckWorkbook(t *testing.T, b Book, q Quirks) {
	t.Helper()
	if got, want := b.Sheets(), []string{"Daten", "Zweites Blatt", "Leer", "Viele"}; !slices.Equal(got, want) {
		t.Fatalf("sheets = %q, want %q", got, want)
	}
	header, rows := Sheet(t, b, "")
	if !slices.Equal(Trim(header), DatenHeader) {
		t.Errorf("header = %q", header)
	}
	if want := Daten(q); !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("Daten:\n got %q\nwant %q", rows, want)
	}
	header, rows = Sheet(t, b, "2")
	if !slices.Equal(Trim(header), []string{"x", "y"}) || !slices.EqualFunc(rows, [][]string{{"1", "2"}, {"2.5", "5"}}, slices.Equal) {
		t.Errorf("Zweites Blatt = %q %q", header, rows)
	}
	header, rows = Sheet(t, b, "leer")
	if len(header) != 0 || len(rows) != 0 {
		t.Errorf("Leer = %q %q", header, rows)
	}
	_, rows = Sheet(t, b, "Viele")
	if len(rows) != 1200 {
		t.Fatalf("Viele has %d rows", len(rows))
	}
	for i, row := range rows {
		if want := Viele(i + 1); !slices.Equal(row, want) {
			t.Fatalf("Viele row %d = %q, want %q", i+1, row, want)
		}
	}
	if _, err := b.Rows("nope"); err == nil {
		t.Error("unknown sheet: want error")
	}
}

// Sheet reads a whole sheet with trailing empty cells removed.
func Sheet(t *testing.T, b Book, sheet string) ([]string, [][]string) {
	t.Helper()
	r, err := b.Rows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	return r.Header(), ReadAll(t, r)
}

// ReadAll reads the remaining rows with trailing empty cells removed.
func ReadAll(t testing.TB, r table.Rows) [][]string {
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
		all = append(all, Trim(slices.Clone(row)))
	}
}

// Trim drops trailing empty cells; writers differ in whether they store them.
func Trim(row []string) []string {
	for len(row) > 0 && row[len(row)-1] == "" {
		row = row[:len(row)-1]
	}
	if row == nil {
		return []string{}
	}
	return row
}

// File opens a fixture for reading at offsets; it is closed with the test.
func File(t testing.TB, name string) (*os.File, int64) {
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
	return f, st.Size()
}
