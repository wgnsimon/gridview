package sample

import (
	"bytes"
	"encoding/csv"
	"slices"
	"testing"
)

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, 50); err != nil {
		t.Fatal(err)
	}
	recs, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(recs[0], Header) || len(recs) != 51 {
		t.Fatalf("got %d records, header %q", len(recs), recs[0])
	}
	if !slices.EqualFunc(recs[1:], Rows(50), slices.Equal) {
		t.Error("CSV rows differ from Rows")
	}
}

func TestDeterministic(t *testing.T) {
	if !slices.EqualFunc(Rows(20), Rows(20), slices.Equal) {
		t.Error("two runs differ")
	}
}

func TestDate(t *testing.T) {
	for serial, want := range map[int]string{43831: "2020-01-01", 43890: "2020-02-29", 45322: "2024-01-31", 61: "1900-03-01"} {
		if got := date(serial); got != want {
			t.Errorf("date(%d) = %s, want %s", serial, got, want)
		}
	}
}
