package xls

import (
	"bytes"
	"os"
	"testing"

	"gridview/internal/reader/readertest"
)

func TestFixtures(t *testing.T) {
	quirks := map[string]readertest.Quirks{
		"types.xls":       readertest.Excel,
		"libreoffice.xls": {LeapDay: "1900-02-29", Formula: "0"},
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

func TestRK(t *testing.T) {
	tests := []struct {
		v    uint32
		want float64
	}{
		{42<<2 | 2, 42},
		{0xFFFFFFE6, -7},
		{1234<<2 | 3, 12.34},
		{0x3FF00000 | 1, 0.01},
	}
	for _, tt := range tests {
		if got := rk(tt.v); got != tt.want {
			t.Errorf("rk(%#x) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestStrReader(t *testing.T) {
	tests := []struct {
		name string
		segs [][]byte
		want []string
	}{
		{"compressed then utf16 after continue", [][]byte{{6, 0, 0, 'a', 'b'}, {1, 'c', 0, 'd', 0, 'e', 0, 'f', 0}}, []string{"abcdef"}},
		{"rich runs across continue", [][]byte{{2, 0, 8, 1, 0, 'h', 'i', 1, 2}, {3, 4}, {1, 0, 0, 'x'}}, []string{"hi", "x"}},
		{"surrogate pair across continue", [][]byte{{3, 0, 1, 0x3D, 0xD8}, {1, 0x00, 0xDE, 'x', 0}}, []string{"😀x"}},
		{"truncated", [][]byte{{5, 0, 0, 'a'}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &strReader{segs: tt.segs}
			for _, want := range tt.want {
				if got, ok := r.str(false); !ok || got != want {
					t.Fatalf("got %q, %v, want %q", got, ok, want)
				}
			}
			if tt.want == nil {
				if _, ok := r.str(false); ok {
					t.Error("want failure")
				}
			}
		})
	}
}

func FuzzOpen(f *testing.F) {
	for _, name := range []string{"../testdata/types.xls", "../testdata/libreoffice.xls"} {
		data, err := os.ReadFile(name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		b, err := Open(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return
		}
		for _, name := range b.Sheets() {
			if r, err := b.Rows(name); err == nil {
				readertest.ReadAll(t, r)
			}
		}
	})
}
