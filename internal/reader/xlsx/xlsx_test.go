package xlsx

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"gridview/internal/reader/excel"
	"gridview/internal/reader/readertest"
	"gridview/internal/sample"
)

func TestFixtures(t *testing.T) {
	quirks := map[string]readertest.Quirks{
		"types.xlsx":       readertest.Excel,
		"libreoffice.xlsx": {LeapDay: "1900-02-28", Formula: readertest.Excel.Formula},
	}
	for file, q := range quirks {
		t.Run(file, func(t *testing.T) {
			b, err := Open(readertest.File(t, "../testdata/"+file))
			if err != nil {
				t.Fatal(err)
			}
			readertest.CheckWorkbook(t, b, q)
		})
	}
}

func TestSampleWorkbook(t *testing.T) {
	var buf bytes.Buffer
	if err := sample.WriteXLSX(&buf, 300); err != nil {
		t.Fatal(err)
	}
	b, err := Open(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	header, rows := readertest.Sheet(t, b, "")
	want := sample.Rows(300)
	for i := range want {
		want[i] = readertest.Trim(want[i])
	}
	if !slices.Equal(header, sample.Header) || !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("sample workbook differs, first row %q, want %q", rows[0], want[0])
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

func FuzzSheet(f *testing.F) {
	f.Add(`<sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1" t="inlineStr"><is><t>x&amp;y</t></is></c></row><row r="3"/></sheetData>`)
	f.Add(`<row><c t="b"><v>1</v></c><c s="1"><v>45322.5</v></c><c t="e"><v>#N/A</v></c></row>`)
	f.Add(`<row r="2"><c r="ZZZZZZ2"><v>1</v></c><c r="B2" t="str"><f>A1</f><v><![CDATA[<x>]]></v></c></row>`)
	book := &Book{strings: []string{"s0"}, formats: []excel.Format{{}, {Kind: excel.Date}}}
	f.Fuzz(func(t *testing.T, xml string) {
		r, err := newRows(book, io.NopCloser(strings.NewReader(xml)))
		if err != nil {
			return
		}
		for range 10000 {
			if _, err := r.Next(); err != nil {
				return
			}
		}
	})
}
