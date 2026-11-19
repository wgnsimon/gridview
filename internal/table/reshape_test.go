package table

import (
	"io"
	"slices"
	"testing"
)

func collect(t *testing.T, r Rows) [][]string {
	t.Helper()
	var out [][]string
	for {
		row, err := r.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, slices.Clone(row))
	}
}

func TestReshape(t *testing.T) {
	src := [][]string{{"note"}, {"id", "name"}, {"1", "a"}, {"2", "b"}}
	tests := []struct {
		skip   int
		header bool
		want   []string
		rows   int
		first  []string
	}{
		{0, true, []string{"note"}, 3, []string{"id", "name"}},
		{1, true, []string{"id", "name"}, 2, []string{"1", "a"}},
		{1, false, []string{"A", "B"}, 3, []string{"id", "name"}},
		{0, false, []string{"A"}, 4, []string{"note"}},
		{3, true, []string{"2", "b"}, 0, nil},
		{4, true, nil, 0, nil},
		{9, false, nil, 0, nil},
	}
	for _, tt := range tests {
		r := Reshape(FromSlices(src[0], src[1:]), tt.skip, tt.header)
		if got := r.Header(); !slices.Equal(got, tt.want) {
			t.Errorf("skip %d header %v: header %q, want %q", tt.skip, tt.header, got, tt.want)
		}
		rows := collect(t, r)
		if len(rows) != tt.rows {
			t.Errorf("skip %d header %v: %d rows, want %d", tt.skip, tt.header, len(rows), tt.rows)
		}
		if len(rows) > 0 && !slices.Equal(rows[0], tt.first) {
			t.Errorf("skip %d header %v: first row %q, want %q", tt.skip, tt.header, rows[0], tt.first)
		}
	}
}

func TestReshapeKeepsRows(t *testing.T) {
	r := FromSlices([]string{"a"}, nil)
	if Reshape(r, 0, true) != r {
		t.Error("Reshape without changes should return rows as they are")
	}
}

func TestColumnName(t *testing.T) {
	for i, want := range map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 701: "ZZ", 702: "AAA"} {
		if got := ColumnName(i); got != want {
			t.Errorf("ColumnName(%d) = %q, want %q", i, got, want)
		}
	}
}
