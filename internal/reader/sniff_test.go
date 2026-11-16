package reader

import "testing"

func TestSniffDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		complete bool
		want     rune
	}{
		{"comma", "a,b,c\n1,2,3\n", true, ','},
		{"semicolon with decimal commas", "a;b\n1,5;2,5\n3,25;4\n", true, ';'},
		{"german decimals everywhere", "x;y\n1,5;2,5\n3,5;4,5\n", true, ';'},
		{"tab", "a\tb\n1\t2\n", true, '\t'},
		{"pipe", "a|b\n1|2\n", true, '|'},
		{"quoted delimiters ignored", "a;b\n\"x,y,z\";1\n\"p,q\";2\n", true, ';'},
		{"newline in quotes", "a;b\n\"x\ny\";1\n", true, ';'},
		{"single column", "name\nAnna\n", true, ','},
		{"header only", "a;b", true, ';'},
		{"crlf", "a;b\r\n1;2\r\n", true, ';'},
		{"inconsistent picks most lines", "a;b\nx;y;z\n1,2\n", true, ';'},
		{"partial last line ignored", "a;b\n1;2\n3,4,5,6,7", false, ';'},
		{"empty", "", true, ','},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sniffDelimiter([]byte(tt.in), tt.complete); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
