package xlsx

import (
	"bytes"
	"slices"
	"testing"

	"gridview/internal/sample"
)

func TestSampleWorkbook(t *testing.T) {
	var buf bytes.Buffer
	if err := sample.WriteXLSX(&buf, 300); err != nil {
		t.Fatal(err)
	}
	b, err := Open(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	header, rows := readSheet(t, b, "")
	if !slices.Equal(header, sample.Header) {
		t.Errorf("header = %q", header)
	}
	want := sample.Rows(300)
	for i := range want {
		want[i] = trim(want[i])
	}
	if !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("rows differ, first got %q want %q", rows[0], want[0])
	}
}
