package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

const fixtures = "../../internal/reader/testdata/"

// TestGolden runs grid end to end on every fixture and compares the output.
func TestGolden(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
	}{
		{"simple", []string{fixtures + "simple.csv"}, ""},
		{"quoted", []string{fixtures + "quoted.csv"}, ""},
		{"ragged", []string{fixtures + "ragged.csv"}, ""},
		{"excel-local", []string{fixtures + "excel-local.csv"}, ""},
		{"excel-utf8", []string{fixtures + "excel-utf8.csv"}, ""},
		{"excel-unicode", []string{fixtures + "excel-unicode.txt"}, ""},
		{"types-xlsx", []string{fixtures + "types.xlsx"}, ""},
		{"types-xlsx-sheet2", []string{"-s", "2", fixtures + "types.xlsx"}, ""},
		{"types-xlsx-empty", []string{fixtures + "types.xlsx", "-sheet", "Leer"}, ""},
		{"types-xls", []string{fixtures + "types.xls"}, ""},
		{"libreoffice-xlsx", []string{fixtures + "libreoffice.xlsx"}, ""},
		{"libreoffice-xls", []string{fixtures + "libreoffice.xls"}, ""},
		{"types-xlsx-width-60", []string{"-w", "60", fixtures + "types.xlsx"}, ""},
		{"simple-width-12", []string{fixtures + "simple.csv", "-w", "12"}, ""},
		{"delimiter-flag", []string{"-d", "tab", fixtures + "excel-unicode.txt"}, ""},
		{"stdin", nil, "a,b\n1,2\n"},
		{"stdin-dash-semicolon", []string{"-"}, "a;b\n1;2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(tt.args, strings.NewReader(tt.stdin), &out, io.Discard); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", tt.name+".golden")
			if *update {
				if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out.Bytes(), want) {
				t.Errorf("output differs from %s, run go test -update to see the diff in git:\n%s", path, out.String())
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	for _, args := range [][]string{
		{fixtures + "missing.csv"},
		{"-s", "nope", fixtures + "types.xlsx"},
		{"a.csv", "b.csv"},
	} {
		if err := run(args, strings.NewReader(""), io.Discard, io.Discard); err == nil {
			t.Errorf("%q: want error", args)
		}
	}
}
