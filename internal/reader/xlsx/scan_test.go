package xlsx

import (
	"io"
	"strings"
	"testing"
)

type token struct {
	text, name string
	kind       tagKind
}

func scanAll(t *testing.T, in string, size int) []token {
	t.Helper()
	sc := newScanner(strings.NewReader(in), size)
	var out []token
	for {
		text, err := sc.text()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		plain := string(unescape(nil, text))
		tag, err := sc.tag()
		if err != nil {
			t.Fatal(err)
		}
		name, kind := parseTag(tag)
		if c, ok := cdata(tag); ok {
			name = []byte("CDATA:" + string(c))
		}
		out = append(out, token{plain, string(name), kind})
	}
}

func TestScanner(t *testing.T) {
	in := `<?xml version="1.0"?><x:row r="1"><!-- a > b --><c a="1>2" b='x"y'/>a&amp;b&#228;&#x1F600;&bogus;` +
		"\r\nz&#13;<![CDATA[<keep>]]></x:row>"
	want := []token{
		{"", "", otherTag},
		{"", "row", startTag},
		{"", "", otherTag},
		{"", "c", emptyTag},
		{"a&bä😀&bogus;\nz\r", "CDATA:<keep>", otherTag},
		{"", "row", endTag},
	}
	for _, size := range []int{16, 4096} {
		got := scanAll(t, in, size)
		if len(got) != len(want) {
			t.Fatalf("size %d: got %d tokens %v", size, len(got), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("size %d token %d = %+v, want %+v", size, i, got[i], want[i])
			}
		}
	}
}

func TestAttrs(t *testing.T) {
	got := map[string]string{}
	attrs([]byte(`c r="B2" xr:uid='{1}' s = "3" t="s"`), func(k, v []byte) {
		got[string(k)] = string(v)
	})
	want := map[string]string{"r": "B2", "uid": "{1}", "s": "3", "t": "s"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestUnterminatedTag(t *testing.T) {
	sc := newScanner(strings.NewReader(`<row r="1`), 16)
	if _, err := sc.text(); err != nil {
		t.Fatal(err)
	}
	if _, err := sc.tag(); err == nil {
		t.Error("want error")
	}
}

func TestNumberCells(t *testing.T) {
	tests := map[string]string{"42": "42", "-3.5": "-3.5", "1.50": "1.5", "2.0": "2", "0.30000000000000004": "0.3", "1E+20": "1e+20", "abc": "abc"}
	r := &rows{b: &Book{}}
	for in, want := range tests {
		r.val = []byte(in)
		if got := string(r.number(nil, 0)); got != want {
			t.Errorf("number(%q) = %q, want %q", in, got, want)
		}
	}
}
