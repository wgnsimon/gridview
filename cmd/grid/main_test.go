package main

import (
	"io"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args []string
		want config
	}{
		{[]string{"a.csv"}, config{path: "a.csv", width: -1}},
		{[]string{"-v", "-s", "2", "a.xlsx", "-y"}, config{path: "a.xlsx", view: true, sheet: "2", width: -1, yes: true}},
		{[]string{"-d", "tab", "a.csv", "-w", "80"}, config{path: "a.csv", delim: '\t', width: 80}},
	}
	for _, tt := range tests {
		got, err := parseArgs(tt.args, io.Discard)
		if err != nil {
			t.Errorf("%q: %v", tt.args, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%q: got %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestParseArgsErrors(t *testing.T) {
	for _, args := range [][]string{{"a", "b"}, {"-d", ";;"}, {"-d", `"`}} {
		if _, err := parseArgs(args, io.Discard); err == nil {
			t.Errorf("%q: want error", args)
		}
	}
}
