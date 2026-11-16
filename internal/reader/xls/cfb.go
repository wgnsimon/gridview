// Package xls reads Excel 97-2003 workbooks (BIFF8 inside a Compound File).
package xls

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf16"
)

const (
	endOfChain = 0xFFFFFFFE
	freeSect   = 0xFFFFFFFF
	dirSize    = 128
	headerSize = 512
)

var errCorrupt = errors.New("xls: corrupt compound file")

// cfb is a Compound File Binary container, read fully into memory sector tables.
type cfb struct {
	r         io.ReaderAt
	size      int64
	shift     uint
	miniShift uint
	cutoff    uint32
	fat       []uint32
	miniFat   []uint32
	dir       []entry
}

type entry struct {
	name  string
	typ   byte
	start uint32
	size  uint64
}

func openCFB(r io.ReaderAt, size int64) (*cfb, error) {
	h := make([]byte, headerSize)
	if _, err := r.ReadAt(h, 0); err != nil {
		return nil, errCorrupt
	}
	le := binary.LittleEndian
	c := &cfb{r: r, size: size, shift: uint(le.Uint16(h[30:])), miniShift: uint(le.Uint16(h[32:])), cutoff: le.Uint32(h[56:])}
	if c.shift != 9 && c.shift != 12 || c.miniShift != 6 {
		return nil, errCorrupt
	}
	if err := c.readFAT(h); err != nil {
		return nil, err
	}
	dir, err := c.chain(le.Uint32(h[48:]), 0)
	if err != nil {
		return nil, err
	}
	for i := 0; i+dirSize <= len(dir); i += dirSize {
		e := parseEntry(dir[i : i+dirSize])
		if c.shift == 9 {
			e.size &= 0xFFFFFFFF
		}
		c.dir = append(c.dir, e)
	}
	if len(c.dir) == 0 || c.dir[0].typ != 5 {
		return nil, errCorrupt
	}
	if n := le.Uint32(h[64:]); n > 0 {
		mf, err := c.chain(le.Uint32(h[60:]), int(n)<<c.shift)
		if err != nil {
			return nil, err
		}
		c.miniFat = u32s(mf)
	}
	return c, nil
}

func (c *cfb) sectorSize() int {
	return 1 << c.shift
}

func (c *cfb) readFAT(h []byte) error {
	le := binary.LittleEndian
	n := int(le.Uint32(h[44:]))
	if n < 0 || int64(n) > c.size>>c.shift+1 {
		return errCorrupt
	}
	sectors := make([]uint32, 0, n)
	for i := 0; i < 109 && len(sectors) < n; i++ {
		sectors = append(sectors, le.Uint32(h[76+4*i:]))
	}
	next := le.Uint32(h[68:])
	per := c.sectorSize()/4 - 1
	for seen := 0; len(sectors) < n && next != endOfChain && next != freeSect; seen++ {
		if seen > n {
			return errCorrupt
		}
		buf, err := c.sector(next)
		if err != nil {
			return err
		}
		ids := u32s(buf)
		for i := 0; i < per && len(sectors) < n; i++ {
			sectors = append(sectors, ids[i])
		}
		next = ids[per]
	}
	c.fat = make([]uint32, 0, n*c.sectorSize()/4)
	for _, s := range sectors {
		buf, err := c.sector(s)
		if err != nil {
			return err
		}
		c.fat = append(c.fat, u32s(buf)...)
	}
	return nil
}

func (c *cfb) sector(n uint32) ([]byte, error) {
	off := (int64(n) + 1) << c.shift
	if off < 0 || off >= c.size {
		return nil, errCorrupt
	}
	buf := make([]byte, c.sectorSize())
	if _, err := c.r.ReadAt(buf, off); err != nil && err != io.EOF {
		return nil, err
	}
	return buf, nil
}

// chain reads a sector chain; size is a capacity hint.
func (c *cfb) chain(start uint32, size int) ([]byte, error) {
	ss := c.sectorSize()
	out := make([]byte, 0, min(size, int(c.size))+ss)
	for n, steps := start, 0; n != endOfChain; steps++ {
		if int(n) >= len(c.fat) || steps > len(c.fat) {
			return nil, errCorrupt
		}
		off := (int64(n) + 1) << c.shift
		if off < 0 || off >= c.size {
			return nil, errCorrupt
		}
		k := len(out)
		out = slices.Grow(out, ss)[:k+ss]
		if m, err := c.r.ReadAt(out[k:], off); err != nil && (err != io.EOF || m == 0) {
			return nil, errCorrupt
		}
		n = c.fat[n]
	}
	return out, nil
}

func (c *cfb) miniChain(start uint32, size uint64) ([]byte, error) {
	stream, err := c.chain(c.dir[0].start, int(min(c.dir[0].size, uint64(c.size))))
	if err != nil {
		return nil, err
	}
	mini := 1 << c.miniShift
	var out []byte
	for n, steps := start, 0; n != endOfChain && uint64(len(out)) < size; steps++ {
		if int(n) >= len(c.miniFat) || steps > len(c.miniFat) {
			return nil, errCorrupt
		}
		off := int(n) * mini
		if off+mini > len(stream) {
			return nil, errCorrupt
		}
		out = append(out, stream[off:off+mini]...)
		n = c.miniFat[n]
	}
	return out, nil
}

// stream returns the content of the first stream with one of the given names.
func (c *cfb) stream(names ...string) ([]byte, string, error) {
	for _, want := range names {
		for _, e := range c.dir {
			if e.typ != 2 || !strings.EqualFold(e.name, want) {
				continue
			}
			var data []byte
			var err error
			if e.size < uint64(c.cutoff) {
				data, err = c.miniChain(e.start, e.size)
			} else {
				data, err = c.chain(e.start, int(min(e.size, uint64(c.size))))
			}
			if err != nil {
				return nil, "", err
			}
			if uint64(len(data)) < e.size {
				return nil, "", errCorrupt
			}
			return data[:e.size], e.name, nil
		}
	}
	return nil, "", fmt.Errorf("xls: no %s stream", strings.Join(names, " or "))
}

func parseEntry(b []byte) entry {
	le := binary.LittleEndian
	n := int(le.Uint16(b[64:]))/2 - 1
	n = max(0, min(n, 31))
	u := make([]uint16, n)
	for i := range u {
		u[i] = le.Uint16(b[2*i:])
	}
	return entry{
		name:  string(utf16.Decode(u)),
		typ:   b[66],
		start: le.Uint32(b[116:]),
		size:  le.Uint64(b[120:]),
	}
}

func u32s(b []byte) []uint32 {
	out := make([]uint32, len(b)/4)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(b[4*i:])
	}
	return out
}
