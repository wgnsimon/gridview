package reader

import (
	"bufio"
	"encoding/binary"
	"io"
	"unicode/utf16"
	"unicode/utf8"
)

const peekSize = 64 << 10

// utf8Text returns a reader that yields UTF-8. It strips a BOM and converts
// UTF-16 and, if the start is not valid UTF-8, Windows-1252.
func utf8Text(br *bufio.Reader) *bufio.Reader {
	head, _ := br.Peek(3)
	switch {
	case len(head) >= 3 && head[0] == 0xEF && head[1] == 0xBB && head[2] == 0xBF:
		br.Discard(3)
		return br
	case len(head) >= 2 && head[0] == 0xFF && head[1] == 0xFE:
		br.Discard(2)
		return convert(br, utf16Conv(binary.LittleEndian))
	case len(head) >= 2 && head[0] == 0xFE && head[1] == 0xFF:
		br.Discard(2)
		return convert(br, utf16Conv(binary.BigEndian))
	}
	sample, _ := br.Peek(peekSize)
	if looksUTF8(sample) {
		return br
	}
	return convert(br, cp1252Conv)
}

func looksUTF8(b []byte) bool {
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				b = b[:i]
			}
			break
		}
	}
	return utf8.Valid(b)
}

// conv converts src to UTF-8 and returns how many bytes of src it used.
type conv func(dst, src []byte, eof bool) ([]byte, int)

type convReader struct {
	r    io.Reader
	conv conv
	raw  []byte
	n    int
	out  []byte
	off  int
	err  error
}

func convert(r io.Reader, c conv) *bufio.Reader {
	return bufio.NewReaderSize(&convReader{r: r, conv: c, raw: make([]byte, 32<<10)}, peekSize)
}

func (c *convReader) Read(p []byte) (int, error) {
	for c.off == len(c.out) {
		if c.err != nil {
			if c.n == 0 {
				return 0, c.err
			}
			c.out, _ = c.conv(c.out[:0], c.raw[:c.n], true)
			c.off, c.n = 0, 0
			continue
		}
		m, err := c.r.Read(c.raw[c.n:])
		c.n += m
		c.err = err
		var used int
		c.out, used = c.conv(c.out[:0], c.raw[:c.n], false)
		c.off = 0
		c.n = copy(c.raw, c.raw[used:c.n])
	}
	n := copy(p, c.out[c.off:])
	c.off += n
	return n, nil
}

func utf16Conv(order binary.ByteOrder) conv {
	return func(dst, src []byte, eof bool) ([]byte, int) {
		i := 0
		for ; i+1 < len(src); i += 2 {
			r := rune(order.Uint16(src[i:]))
			if utf16.IsSurrogate(r) {
				if i+3 >= len(src) && !eof {
					break
				}
				if r < 0xDC00 && i+3 < len(src) {
					if d := utf16.DecodeRune(r, rune(order.Uint16(src[i+2:]))); d != utf8.RuneError {
						dst = utf8.AppendRune(dst, d)
						i += 2
						continue
					}
				}
				r = utf8.RuneError
			}
			dst = utf8.AppendRune(dst, r)
		}
		if eof {
			i = len(src)
		}
		return dst, i
	}
}

func cp1252Conv(dst, src []byte, _ bool) ([]byte, int) {
	for _, b := range src {
		switch {
		case b < 0x80:
			dst = append(dst, b)
		case b < 0xA0:
			dst = utf8.AppendRune(dst, cp1252[b-0x80])
		default:
			dst = utf8.AppendRune(dst, rune(b))
		}
	}
	return dst, len(src)
}

var cp1252 = [32]rune{
	0x20AC, 0xFFFD, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0xFFFD, 0x017D, 0xFFFD,
	0xFFFD, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0xFFFD, 0x017E, 0x0178,
}
