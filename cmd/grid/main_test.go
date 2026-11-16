package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args []string
		want config
	}{
		{[]string{"a.csv"}, config{path: "a.csv", width: -1}},
		{[]string{"a.csv", "-view"}, config{path: "a.csv", view: true, width: -1}},
		{[]string{"-v", "-s", "2", "a.xlsx"}, config{path: "a.xlsx", view: true, sheet: "2", width: -1}},
		{[]string{"-d", ";", "a.csv", "-w", "80"}, config{path: "a.csv", delim: ';', width: 80}},
		{[]string{"-d", "tab"}, config{delim: '\t', width: -1}},
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
	for _, args := range [][]string{{"a", "b"}, {"-d", ";;"}, {"-d", `"`}, {"-nope"}} {
		if _, err := parseArgs(args, io.Discard); err == nil {
			t.Errorf("%q: want error", args)
		}
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"file", []string{"../../internal/reader/testdata/simple.csv"}, "", "id | name | city\n---+------+-------\n 1 | Anna | Berlin\n 2 | Ömer | Köln\n"},
		{"stdin", nil, "a,b\n1,2\n", "a | b\n--+--\n1 | 2\n"},
		{"dash", []string{"-", "-w", "0"}, "a;b\n1;2\n", "a;b\n---\n1;2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(tt.args, strings.NewReader(tt.stdin), &out, io.Discard); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", out.String(), tt.want)
			}
		})
	}
}
