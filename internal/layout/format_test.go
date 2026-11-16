package layout

import "testing"

func TestAppendRow(t *testing.T) {
	tests := []struct {
		name string
		f    Formatter
		row  []string
		want string
	}{
		{"pads all but last", Formatter{Cols: []Column{{2, false}, {4, false}}, Sep: " | "}, []string{"1", "Bo"}, "1  | Bo"},
		{"right aligned", Formatter{Cols: []Column{{4, true}, {1, false}}, Sep: " "}, []string{"42", "x"}, "  42 x"},
		{"missing cells", Formatter{Cols: []Column{{1, false}, {2, false}, {1, false}}, Sep: "|"}, []string{"a"}, "a|  |"},
		{"cut wide rune", Formatter{Cols: []Column{{4, false}}, Sep: "|", Cut: true}, []string{"田中太郎"}, "田… "},
		{"control chars", Formatter{Cols: []Column{{5, false}}}, []string{"a\nb\tc"}, "a↵b c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.f.AppendRow(nil, tt.row)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
