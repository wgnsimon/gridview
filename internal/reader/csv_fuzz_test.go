package reader

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"slices"
	"testing"
	"unicode/utf8"
)

var csvSeeds = []string{
	"a,b\n1,2\n",
	"a,b\r\n1,2\r\n",
	"a,\"b,c\"\n\"x\"\"y\",z\n",
	"\"multi\nline\",x\n\"two\r\nlines\",y",
	"a\"b,c\n\"a\"b,c\n\"a\"\"\n",
	"\n\n a , b \n\r\n,\n,,\n",
	"\"unterminated,x\nnext",
	"trailing\r",
	"\"\"\"\",\"\"\n",
	"a;b\"c;\"d;e\";f\n",
	"x\ry\rz\n",
	"\"a\"\r\n\"b\"\r\n",
	"\"x\" ,y\n",
}

func FuzzCSVMatchesEncodingCSV(f *testing.F) {
	for _, s := range csvSeeds {
		f.Add(s, byte(','))
		f.Add(s, byte(';'))
	}
	f.Fuzz(func(t *testing.T, in string, delim byte) {
		r := rune(delim)
		if r == '"' || r == '\r' || r == '\n' || r == 0 || r >= utf8.RuneSelf {
			return
		}
		want, wantErr := encodingCSV(in, r)
		got, gotErr := ourCSV(in, r)
		if (wantErr != nil) != (gotErr != nil) {
			t.Fatalf("%q delim %q: error %v, encoding/csv %v", in, r, gotErr, wantErr)
		}
		if wantErr == nil && !slices.EqualFunc(got, want, slices.Equal) {
			t.Fatalf("%q delim %q:\n got %q\nwant %q", in, r, got, want)
		}
	})
}

func encodingCSV(in string, delim rune) ([][]string, error) {
	r := csv.NewReader(bytes.NewBufferString(in))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	return r.ReadAll()
}

func ourCSV(in string, delim rune) ([][]string, error) {
	c, err := newCSV(bufio.NewReaderSize(bytes.NewBufferString(in), 16), delim)
	if err != nil {
		return nil, err
	}
	var all [][]string
	if h := c.Header(); h != nil {
		all = append(all, h)
	}
	for {
		row, err := c.Next()
		if err == io.EOF {
			return all, nil
		}
		if err != nil {
			return nil, err
		}
		all = append(all, slices.Clone(row))
	}
}
