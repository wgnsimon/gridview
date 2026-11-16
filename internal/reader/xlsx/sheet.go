package xlsx

import (
	"bytes"
	"io"
	"slices"
	"strconv"
	"unsafe"

	"gridview/internal/reader/excel"
)

const scanSize = 64 << 10

type rows struct {
	b      *Book
	rc     io.ReadCloser
	sc     *scanner
	header []string
	cells  []string
	parts  []part
	line   []byte
	val    []byte
	last   int
	next   int
	ready  bool
	err    error
}

// part is a finished cell: a ready string or a range of the row buffer.
type part struct {
	s          string
	start, end int
	inLine     bool
}

func newRows(b *Book, rc io.ReadCloser) (*rows, error) {
	r := &rows{b: b, rc: rc, sc: newScanner(rc, scanSize)}
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
	num, col := 0, -1
	typ, style := byte('n'), 0
	inRow, inValue, skip := false, false, 0
	for {
		text, err := r.sc.text()
		if err != nil {
			return 0, err
		}
		if inValue && skip == 0 {
			r.val = unescape(r.val, text)
		}
		tag, err := r.sc.tag()
		if err != nil {
			return 0, err
		}
		name, kind := parseTag(tag)
		if kind == otherTag {
			if c, ok := cdata(tag); ok && inValue && skip == 0 {
				r.val = append(r.val, c...)
			}
			continue
		}
		switch string(name) {
		case "row":
			switch kind {
			case endTag:
				if inRow {
					return num, r.finishRow()
				}
			default:
				inRow = true
				r.parts, r.line = r.parts[:0], r.line[:0]
				num, col = r.last+1, -1
				attrs(tag, func(k, v []byte) {
					if len(k) == 1 && k[0] == 'r' {
						if n := atoi(v); n > 0 {
							num = n
						}
					}
				})
				if kind == emptyTag {
					return num, r.finishRow()
				}
			}
		case "c":
			if kind == endTag {
				r.finishCell(col, typ, style)
				continue
			}
			col++
			typ, style = 'n', 0
			attrs(tag, func(k, v []byte) {
				if len(k) != 1 {
					return
				}
				switch k[0] {
				case 'r':
					if c := column(unsafe.String(unsafe.SliceData(v), len(v))); c >= 0 {
						col = c
					}
				case 't':
					typ = cellType(v)
				case 's':
					style = atoi(v)
				}
			})
			r.val = r.val[:0]
			if kind == emptyTag {
				r.finishCell(col, typ, style)
			}
		case "v", "t":
			inValue = inRow && kind == startTag
		case "rPh":
			if kind == startTag {
				skip++
			} else if kind == endTag {
				skip--
			}
		case "sheetData":
			if kind == endTag {
				return 0, io.EOF
			}
		}
	}
}

// cellType maps the t attribute to one byte, the buffer behind v is reused.
func cellType(v []byte) byte {
	switch string(v) {
	case "s", "b", "n", "e", "d":
		return v[0]
	case "str":
		return 't'
	case "inlineStr":
		return 'i'
	}
	return 'n'
}

func (r *rows) finishCell(col int, typ byte, style int) {
	if col < 0 || col > maxCol {
		return
	}
	for len(r.parts) <= col {
		r.parts = append(r.parts, part{})
	}
	p := &r.parts[col]
	switch typ {
	case 's':
		if i := atoi(r.val); i >= 0 && i < len(r.b.strings) {
			*p = part{s: r.b.strings[i]}
		}
		return
	case 'b':
		*p = part{s: "FALSE"}
		if bytes.Equal(r.val, []byte("1")) {
			p.s = "TRUE"
		}
		return
	case 'n':
		start := len(r.line)
		r.line = r.number(r.line, style)
		*p = part{start: start, end: len(r.line), inLine: true}
		return
	}
	start := len(r.line)
	r.line = append(r.line, r.val...)
	*p = part{start: start, end: len(r.line), inLine: true}
}

func (r *rows) finishRow() error {
	line := string(r.line)
	r.cells = r.cells[:0]
	for _, p := range r.parts {
		if p.inLine {
			p.s = line[p.start:p.end]
		}
		r.cells = append(r.cells, p.s)
	}
	return nil
}

// number appends the cell value formatted with its style.
func (r *rows) number(dst []byte, style int) []byte {
	f := excel.Format{}
	if style >= 0 && style < len(r.b.formats) {
		f = r.b.formats[style]
	}
	if f.Kind == excel.General || f.Kind == excel.Text {
		if plain(r.val) {
			return trimZeros(dst, r.val)
		}
	}
	v, err := strconv.ParseFloat(unsafe.String(unsafe.SliceData(r.val), len(r.val)), 64)
	if err != nil {
		return append(dst, r.val...)
	}
	return excel.AppendNumber(dst, v, f, r.b.dates)
}

// plain reports whether v is a decimal without exponent and at most 15 digits,
// which General shows unchanged.
func plain(v []byte) bool {
	digits := 0
	for i, c := range v {
		switch {
		case c >= '0' && c <= '9':
			digits++
		case c == '-' && i == 0:
		case c == '.':
		default:
			return false
		}
	}
	return digits > 0 && digits <= 15
}

func trimZeros(dst, v []byte) []byte {
	if bytes.IndexByte(v, '.') >= 0 {
		v = bytes.TrimRight(v, "0")
		v = bytes.TrimSuffix(v, []byte("."))
	}
	return append(dst, v...)
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
