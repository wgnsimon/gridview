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

// TestGolden runs grid end to end and compares the output.
func TestGolden(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
	}{
		{"excel-local", []string{fixtures + "excel-local.csv"}, ""},
		{"types-xlsx", []string{fixtures + "types.xlsx"}, ""},
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
