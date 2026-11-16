package xls

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"gridview/internal/reader/excel"
	"gridview/internal/table"
)

const (
	recFormula   = 0x0006
	recEOF       = 0x000A
	recDateMode  = 0x0022
	recFilePass  = 0x002F
	recContinue  = 0x003C
	recBoundSht  = 0x0085
	recMulRK     = 0x00BD
	recRString   = 0x00D6
	recXF        = 0x00E0
	recSST       = 0x00FC
	recLabelSST  = 0x00FD
	recNumber    = 0x0203
	recLabel     = 0x0204
	recBoolErr   = 0x0205
	recString    = 0x0207
	recRK        = 0x027E
	recFormat    = 0x041E
	recBOF       = 0x0809
	biff8Version = 0x0600
)

var le = binary.LittleEndian

// Book is an opened xls workbook.
type Book struct {
	wb      []byte
	names   []string
	offsets []int
	sst     []string
	formats []excel.Format
	dates   excel.DateSystem
}

// Open reads the workbook globals: sheets, shared strings and formats.
func Open(r io.ReaderAt, size int64) (*Book, error) {
	c, err := openCFB(r, size)
	if err != nil {
		return nil, err
	}
	data, name, err := c.stream("Workbook", "Book")
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(name, "Book") {
		return nil, errors.New("xls: Excel 5.0/95 workbooks are not supported")
	}
	b := &Book{wb: data}
	return b, b.readGlobals()
}

// Sheets returns the worksheet names in workbook order.
func (b *Book) Sheets() []string {
	return b.names
}

// Rows reads a sheet chosen by name or 1-based index; empty picks the first.
func (b *Book) Rows(sheet string) (table.Rows, error) {
	i, err := excel.PickSheet(b.names, sheet)
	if err != nil {
		return nil, err
	}
	grid, err := b.readSheet(b.offsets[i])
	if err != nil {
		return nil, err
	}
	first := 0
	for first < len(grid) && grid[first] == nil {
		first++
	}
	if first == len(grid) {
		return table.FromSlices(nil, nil), nil
	}
	return table.FromSlices(grid[first], grid[first+1:]), nil
}

type record struct {
	typ  uint16
	data []byte
}

// records iterates BIFF records starting at off.
type records struct {
	b   []byte
	off int
}

func (it *records) next() (record, bool) {
	if it.off+4 > len(it.b) {
		return record{}, false
	}
	typ, n := le.Uint16(it.b[it.off:]), int(le.Uint16(it.b[it.off+2:]))
	start := it.off + 4
	if start+n > len(it.b) {
		return record{}, false
	}
	it.off = start + n
	return record{typ, it.b[start : start+n]}, true
}

// continues collects the data of a record and the CONTINUE records after it.
func (it *records) continues(first []byte) [][]byte {
	segs := [][]byte{first}
	for {
		save := it.off
		rec, ok := it.next()
		if !ok || rec.typ != recContinue {
			it.off = save
			return segs
		}
		segs = append(segs, rec.data)
	}
}

func (b *Book) readGlobals() error {
	it := &records{b: b.wb}
	rec, ok := it.next()
	if !ok || rec.typ != recBOF || len(rec.data) < 2 || le.Uint16(rec.data) != biff8Version {
		return errors.New("xls: only BIFF8 (Excel 97 and later) is supported")
	}
	custom := map[uint16]string{}
	var xfs []uint16
	for {
		rec, ok := it.next()
		if !ok {
			return errCorrupt
		}
		switch rec.typ {
		case recEOF:
			b.formats = make([]excel.Format, len(xfs))
			for i, id := range xfs {
				if code, ok := custom[id]; ok {
					b.formats[i] = excel.ParseFormat(code)
				} else {
					b.formats[i] = excel.Builtin(int(id))
				}
			}
			return nil
		case recFilePass:
			return errors.New("xls: encrypted workbooks are not supported")
		case recDateMode:
			if len(rec.data) >= 2 && le.Uint16(rec.data) == 1 {
				b.dates = excel.Date1904
			}
		case recFormat:
			if len(rec.data) > 2 {
				s := &strReader{segs: [][]byte{rec.data[2:]}}
				if code, ok := s.str(false); ok {
					custom[le.Uint16(rec.data)] = code
				}
			}
		case recXF:
			if len(rec.data) >= 4 {
				xfs = append(xfs, le.Uint16(rec.data[2:]))
			}
		case recBoundSht:
			if len(rec.data) < 8 || rec.data[5] > 1 {
				continue
			}
			s := &strReader{segs: [][]byte{rec.data[6:]}}
			name, ok := s.str(true)
			if !ok {
				return errCorrupt
			}
			b.names = append(b.names, name)
			b.offsets = append(b.offsets, int(le.Uint32(rec.data)))
		case recSST:
			if len(rec.data) < 8 {
				return errCorrupt
			}
			if err := b.readSST(it.continues(rec.data[8:]), int(le.Uint32(rec.data[4:]))); err != nil {
				return err
			}
		}
	}
}

func (b *Book) readSST(segs [][]byte, n int) error {
	s := &strReader{segs: segs}
	b.sst = make([]string, 0, min(n, 1<<20))
	for range n {
		str, ok := s.str(false)
		if !ok {
			return errCorrupt
		}
		b.sst = append(b.sst, str)
	}
	return nil
}

func (b *Book) readSheet(off int) ([][]string, error) {
	it := &records{b: b.wb, off: off}
	if rec, ok := it.next(); !ok || rec.typ != recBOF {
		return nil, errCorrupt
	}
	var grid [][]string
	set := func(row, col int, s string) {
		for len(grid) <= row {
			grid = append(grid, nil)
		}
		r := grid[row]
		for len(r) <= col {
			r = append(r, "")
		}
		r[col] = s
		grid[row] = r
	}
	pending := -1
	for {
		rec, ok := it.next()
		if !ok {
			return nil, errCorrupt
		}
		d := rec.data
		if rec.typ == recEOF {
			return grid, nil
		}
		if len(d) < 6 {
			continue
		}
		row, col, xf := int(le.Uint16(d)), int(le.Uint16(d[2:])), int(le.Uint16(d[4:]))
		switch rec.typ {
		case recLabelSST:
			if len(d) >= 10 {
				if i := int(le.Uint32(d[6:])); i < len(b.sst) {
					set(row, col, b.sst[i])
				}
			}
		case recNumber:
			if len(d) >= 14 {
				set(row, col, b.number(math.Float64frombits(le.Uint64(d[6:])), xf))
			}
		case recRK:
			if len(d) >= 10 {
				set(row, col, b.number(rk(le.Uint32(d[6:])), xf))
			}
		case recMulRK:
			for k := 0; 6+6*k+6 <= len(d); k++ {
				set(row, col+k, b.number(rk(le.Uint32(d[6+6*k:])), int(le.Uint16(d[4+6*k:]))))
			}
		case recLabel, recRString:
			s := &strReader{segs: [][]byte{d[6:]}}
			if str, ok := s.str(false); ok {
				set(row, col, str)
			}
		case recBoolErr:
			if len(d) >= 8 {
				set(row, col, boolErr(d[6], d[7] != 0))
			}
		case recFormula:
			if len(d) < 14 {
				continue
			}
			v := d[6:14]
			if v[6] != 0xFF || v[7] != 0xFF {
				set(row, col, b.number(math.Float64frombits(le.Uint64(v)), xf))
				continue
			}
			switch v[0] {
			case 0:
				pending = row<<16 | col
			case 1:
				set(row, col, boolErr(v[2], false))
			case 2:
				set(row, col, boolErr(v[2], true))
			default:
				set(row, col, "")
			}
		}
		if rec.typ == recString && pending >= 0 {
			s := &strReader{segs: it.continues(rec.data)}
			if str, ok := s.str(false); ok {
				set(pending>>16, pending&0xFFFF, str)
			}
			pending = -1
		}
	}
}

func (b *Book) number(v float64, xf int) string {
	f := excel.Format{}
	if xf < len(b.formats) {
		f = b.formats[xf]
	}
	return excel.FormatNumber(v, f, b.dates)
}

func rk(v uint32) float64 {
	var f float64
	if v&2 != 0 {
		f = float64(int32(v) >> 2)
	} else {
		f = math.Float64frombits(uint64(v&^3) << 32)
	}
	if v&1 != 0 {
		f /= 100
	}
	return f
}

func boolErr(v byte, isErr bool) string {
	if !isErr {
		if v != 0 {
			return "TRUE"
		}
		return "FALSE"
	}
	switch v {
	case 0x00:
		return "#NULL!"
	case 0x07:
		return "#DIV/0!"
	case 0x0F:
		return "#VALUE!"
	case 0x17:
		return "#REF!"
	case 0x1D:
		return "#NAME?"
	case 0x24:
		return "#NUM!"
	case 0x2A:
		return "#N/A"
	}
	return fmt.Sprintf("#ERR%d", v)
}

// strReader reads BIFF8 strings that may be split over CONTINUE records.
type strReader struct {
	segs [][]byte
	i    int
	pos  int
	out  []byte
}

func (r *strReader) u8() (byte, bool) {
	for r.i < len(r.segs) && r.pos >= len(r.segs[r.i]) {
		r.i++
		r.pos = 0
	}
	if r.i >= len(r.segs) {
		return 0, false
	}
	r.pos++
	return r.segs[r.i][r.pos-1], true
}

func (r *strReader) u16() (int, bool) {
	lo, ok1 := r.u8()
	hi, ok2 := r.u8()
	return int(lo) | int(hi)<<8, ok1 && ok2
}

func (r *strReader) skip(n int) bool {
	for n > 0 {
		if r.i >= len(r.segs) {
			return false
		}
		k := min(n, len(r.segs[r.i])-r.pos)
		r.pos += k
		n -= k
		if n > 0 {
			r.i++
			r.pos = 0
		}
	}
	return true
}

// str reads a unicode string; short means a one-byte character count.
func (r *strReader) str(short bool) (string, bool) {
	var cch int
	ok := true
	if short {
		var c byte
		c, ok = r.u8()
		cch = int(c)
	} else {
		cch, ok = r.u16()
	}
	flags, ok2 := r.u8()
	if !ok || !ok2 {
		return "", false
	}
	runs, ext := 0, 0
	if flags&8 != 0 {
		n, ok := r.u16()
		if !ok {
			return "", false
		}
		runs = n
	}
	if flags&4 != 0 {
		var e [4]byte
		for k := range e {
			if e[k], ok = r.u8(); !ok {
				return "", false
			}
		}
		ext = int(le.Uint32(e[:]))
	}
	wide := flags&1 != 0
	r.out = r.out[:0]
	var high rune
	for need := cch; need > 0; {
		if r.i >= len(r.segs) {
			return "", false
		}
		if r.pos >= len(r.segs[r.i]) {
			r.i++
			if r.i >= len(r.segs) || len(r.segs[r.i]) == 0 {
				return "", false
			}
			wide = r.segs[r.i][0]&1 != 0
			r.pos = 1
			continue
		}
		seg := r.segs[r.i][r.pos:]
		if !wide {
			n := min(need, len(seg))
			for _, c := range seg[:n] {
				if c < utf8.RuneSelf {
					r.out = append(r.out, c)
				} else {
					r.out = utf8.AppendRune(r.out, rune(c))
				}
			}
			r.pos += n
			need -= n
			continue
		}
		n := min(need, len(seg)/2)
		if n == 0 {
			r.pos = len(r.segs[r.i])
			continue
		}
		for k := range n {
			u := rune(le.Uint16(seg[2*k:]))
			switch {
			case high != 0 && utf16.IsSurrogate(u) && u >= 0xDC00:
				r.out = utf8.AppendRune(r.out, utf16.DecodeRune(high, u))
				high = 0
				continue
			case high != 0:
				r.out = utf8.AppendRune(r.out, utf8.RuneError)
				high = 0
			}
			switch {
			case u >= 0xD800 && u < 0xDC00:
				high = u
			case utf16.IsSurrogate(u):
				r.out = utf8.AppendRune(r.out, utf8.RuneError)
			default:
				r.out = utf8.AppendRune(r.out, u)
			}
		}
		r.pos += 2 * n
		need -= n
	}
	if high != 0 {
		r.out = utf8.AppendRune(r.out, utf8.RuneError)
	}
	if !r.skip(4*runs + ext) {
		return "", false
	}
	return string(r.out), true
}
