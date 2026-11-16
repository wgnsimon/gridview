package xlsx

import (
	"encoding/xml"
	"io"
	"slices"
	"strconv"

	"gridview/internal/reader/excel"
)

type rows struct {
	b      *Book
	rc     io.ReadCloser
	dec    *xml.Decoder
	header []string
	cells  []string
	text   []byte
	last   int
	next   int
	ready  bool
	err    error
}

func newRows(b *Book, rc io.ReadCloser) (*rows, error) {
	r := &rows{b: b, rc: rc, dec: xml.NewDecoder(rc)}
	n, err := r.readRow()
	switch {
	case err == io.EOF:
		return r, nil
	case err != nil:
		rc.Close()
		return nil, err
	}
	r.header = slices.Clone(r.cells)
	r.last = n
	return r, nil
}

func (r *rows) Header() []string {
	return r.header
}

func (r *rows) Next() ([]string, error) {
	if !r.ready {
		if r.err != nil {
			return nil, r.err
		}
		r.next, r.err = r.readRow()
		if r.err != nil {
			return nil, r.err
		}
		r.ready = true
	}
	if r.next > r.last+1 {
		r.last++
		return nil, nil
	}
	r.ready = false
	r.last = r.next
	return r.cells, nil
}

func (r *rows) Close() error {
	return r.rc.Close()
}

// readRow parses the next <row> into r.cells and returns its 1-based number.
func (r *rows) readRow() (int, error) {
	num, col := 0, 0
	var typ, style string
	inValue, inRow := false, false
	for {
		tok, err := r.dec.Token()
		if err != nil {
			return 0, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				inRow = true
				r.cells = r.cells[:0]
				num = r.last + 1
				col = -1
				if v := attr(t, "r"); v != "" {
					num, _ = strconv.Atoi(v)
				}
			case "c":
				col++
				typ, style = attr(t, "t"), attr(t, "s")
				if c := column(attr(t, "r")); c >= 0 {
					col = c
				}
				r.text = r.text[:0]
			case "v", "t":
				inValue = inRow
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v", "t":
				inValue = false
			case "c":
				if col > maxCol {
					continue
				}
				for len(r.cells) <= col {
					r.cells = append(r.cells, "")
				}
				r.cells[col] = r.value(typ, style)
			case "row":
				return num, nil
			}
		case xml.CharData:
			if inValue {
				r.text = append(r.text, t...)
			}
		}
	}
}

func (r *rows) value(typ, style string) string {
	switch typ {
	case "s":
		i, err := strconv.Atoi(string(r.text))
		if err != nil || i < 0 || i >= len(r.b.strings) {
			return ""
		}
		return r.b.strings[i]
	case "b":
		if string(r.text) == "1" {
			return "TRUE"
		}
		return "FALSE"
	case "", "n":
		v, err := strconv.ParseFloat(string(r.text), 64)
		if err != nil {
			return string(r.text)
		}
		return excel.FormatNumber(v, r.format(style), r.b.dates)
	}
	return string(r.text)
}

func (r *rows) format(style string) excel.Format {
	i, err := strconv.Atoi(style)
	if err != nil || i < 0 || i >= len(r.b.formats) {
		return excel.Format{}
	}
	return r.b.formats[i]
}

func attr(t xml.StartElement, name string) string {
	for _, a := range t.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

const maxCol = 16383

// column turns a cell reference like "AB12" into a 0-based column index, -1 if invalid.
func column(ref string) int {
	n := 0
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c < 'A' || c > 'Z' {
			break
		}
		if n = n*26 + int(c-'A'+1); n > maxCol+1 {
			return maxCol + 1
		}
	}
	return n - 1
}
