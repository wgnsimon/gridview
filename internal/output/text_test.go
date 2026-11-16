package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"gridview/internal/table"
)

func numbers(n int) [][]string {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{fmt.Sprint(i)}
	}
	return rows
}

func TestTextShow(t *testing.T) {
	tests := []struct {
		name     string
		header   []string
		rows     [][]string
		maxWidth int
		border   bool
		want     string
	}{
		{
			name:   "aligns and right aligns numbers",
			header: []string{"id", "name"},
			rows:   [][]string{{"1", "Anna"}, {"22", "Bo"}},
			want:   "id | name\n---+-----\n 1 | Anna\n22 | Bo\n",
		},
		{
			name:     "hides columns that do not fit",
			header:   []string{"aaaaaaaa", "bbbbbbbb", "cccccccc"},
			rows:     [][]string{{"x", "y", "z"}},
			maxWidth: 20,
			want:     "aaaaaaaa | bbbbbbbb\n---------+---------\nx        | y\n(1 more columns, use a wider terminal or -view)\n",
		},
		{
			name:     "frame counts toward max width",
			header:   []string{"id", "name"},
			rows:     [][]string{{"1", "a very long note"}},
			maxWidth: 15,
			border:   true,
			want:     "┌────┬────────┐\n│ id │ name   │\n├────┼────────┤\n│  1 │ a ver… │\n└────┴────────┘\n",
		},
		{
			name: "empty",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			text := Text{W: &buf, MaxWidth: tt.maxWidth, Border: tt.border}
			if err := text.Show(table.FromSlices(tt.header, tt.rows)); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestTextShowStreamsPastSample(t *testing.T) {
	rows := numbers(sampleSize + 5)
	rows[len(rows)-1] = []string{"wider than the sample"}
	var buf bytes.Buffer
	if err := (Text{W: &buf}).Show(table.FromSlices([]string{"n"}, rows)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != len(rows)+2 {
		t.Fatalf("got %d lines, want %d", len(lines), len(rows)+2)
	}
	if last := lines[len(lines)-1]; last != "wider than the sample" {
		t.Errorf("last line = %q", last)
	}
}

func TestTextShowConfirm(t *testing.T) {
	tests := []struct {
		name    string
		rows    int
		answer  bool
		asked   bool
		wantOut bool
	}{
		{name: "at the limit no question", rows: AskAbove, wantOut: true},
		{name: "above the limit yes", rows: AskAbove + 1, answer: true, asked: true, wantOut: true},
		{name: "above the limit no", rows: AskAbove + 1, asked: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asked := false
			var buf bytes.Buffer
			text := Text{W: &buf, Confirm: func(int, bool) bool {
				asked = true
				return tt.answer
			}}
			if err := text.Show(table.FromSlices([]string{"n"}, numbers(tt.rows))); err != nil {
				t.Fatal(err)
			}
			if asked != tt.asked {
				t.Errorf("asked = %v, want %v", asked, tt.asked)
			}
			lines := strings.Count(buf.String(), "\n")
			if tt.wantOut && lines != tt.rows+2 || !tt.wantOut && lines != 0 {
				t.Errorf("printed %d lines, wantOut %v", lines, tt.wantOut)
			}
		})
	}
}
