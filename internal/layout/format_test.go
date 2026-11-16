package layout

import (
	"strings"
	"testing"
)

func TestAppendRow(t *testing.T) {
	tests := []struct {
		name string
		f    Formatter
		row  []string
		want string
	}{
		{"pads all but last", Formatter{Cols: []Column{{2, false}, {4, false}}, Sep: " | "}, []string{"1", "Bo"}, "1  | Bo"},
		{"right aligned", Formatter{Cols: []Column{{4, true}, {1, false}}, Sep: " "}, []string{"42", "x"}, "  42 x"},
		{"right aligned last", Formatter{Cols: []Column{{4, true}}, Sep: " "}, []string{"7"}, "   7"},
		{"umlauts", Formatter{Cols: []Column{{2, false}, {1, false}}, Sep: "|"}, []string{"ä", "x"}, "ä |x"},
		{"missing cells", Formatter{Cols: []Column{{1, false}, {2, false}, {1, false}}, Sep: "|"}, []string{"a"}, "a|  |"},
		{"cut", Formatter{Cols: []Column{{4, false}, {1, false}}, Sep: "|", Cut: true}, []string{"abcdef", "x"}, "abc…|x"},
		{"no cut overflows", Formatter{Cols: []Column{{2, false}, {1, false}}, Sep: "|"}, []string{"abcd", "x"}, "abcd|x"},
		{"cut wide rune", Formatter{Cols: []Column{{4, false}}, Sep: "|", Cut: true}, []string{"田中太郎"}, "田… "},
		{"cut to one", Formatter{Cols: []Column{{1, false}}, Cut: true}, []string{"abc"}, "…"},
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

func TestAppendMarked(t *testing.T) {
	f := Formatter{Cols: []Column{{2, false}, {2, false}}, Sep: "|"}
	if got := string(f.AppendMarked(nil, []string{"a", "b"}, 1, "<", ">")); got != "a |<b>" {
		t.Errorf("got %q", got)
	}
}

func TestAppendRule(t *testing.T) {
	f := Formatter{Cols: []Column{{2, false}, {3, true}}, Sep: " | "}
	if got := string(f.AppendRule(nil)); got != "---+----" {
		t.Errorf("got %q", got)
	}
}

func TestAppendRuleBox(t *testing.T) {
	f := Formatter{Cols: []Column{{2, false}, {3, true}}, Sep: " │ "}
	if got := string(f.AppendRule(nil)); got != "───┼────" {
		t.Errorf("got %q", got)
	}
}

func TestBorder(t *testing.T) {
	f := Formatter{Cols: []Column{{2, false}, {3, true}}, Sep: " │ ", Border: true}
	got := []string{
		string(f.AppendTop(nil)),
		string(f.AppendRow(nil, []string{"a", "1"})),
		string(f.AppendRule(nil)),
		string(f.AppendBottom(nil)),
	}
	want := []string{"┌────┬─────┐", "│ a  │   1 │", "├────┼─────┤", "└────┴─────┘"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
		if w := Width(got[i]); w != 5+3+BorderWidth {
			t.Errorf("line %d is %d wide", i, w)
		}
	}
}

func TestAppendRepeatMultiByte(t *testing.T) {
	if got := string(appendRepeat(nil, lines, 300)); got != strings.Repeat("─", 300) {
		t.Errorf("got %d bytes", len(got))
	}
}
