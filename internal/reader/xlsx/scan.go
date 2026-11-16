package xlsx

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"unicode/utf8"
)

// scanner is a minimal pull parser for the flat XML of sheets and shared strings.
// Returned slices are only valid until the next call.
type scanner struct {
	r   *bufio.Reader
	acc []byte
}

func newScanner(r io.Reader, size int) *scanner {
	return &scanner{r: bufio.NewReaderSize(r, size)}
}

// until reads up to and including delim.
func (s *scanner) until(delim byte) ([]byte, error) {
	b, err := s.r.ReadSlice(delim)
	if err != bufio.ErrBufferFull {
		return b, err
	}
	s.acc = append(s.acc[:0], b...)
	for err == bufio.ErrBufferFull {
		b, err = s.r.ReadSlice(delim)
		s.acc = append(s.acc, b...)
	}
	return s.acc, err
}

// text returns the raw text before the next tag.
func (s *scanner) text() ([]byte, error) {
	b, err := s.until('<')
	if err != nil {
		return nil, err
	}
	return b[:len(b)-1], nil
}

// tag returns what is between < and >. Quotes, comments and CDATA may contain >.
func (s *scanner) tag() ([]byte, error) {
	b, err := s.until('>')
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if complete(b) {
		return b[:len(b)-1], nil
	}
	s.acc = append(s.acc[:0], b...)
	for !complete(s.acc) {
		more, err := s.r.ReadSlice('>')
		s.acc = append(s.acc, more...)
		if err != nil && err != bufio.ErrBufferFull {
			return nil, io.ErrUnexpectedEOF
		}
	}
	return s.acc[:len(s.acc)-1], nil
}

func complete(b []byte) bool {
	switch {
	case bytes.HasPrefix(b, []byte("![CDATA[")):
		return bytes.HasSuffix(b, []byte("]]>"))
	case bytes.HasPrefix(b, []byte("!--")):
		return bytes.HasSuffix(b, []byte("-->"))
	}
	var quote byte
	for _, c := range b {
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		}
	}
	return quote == 0
}

type tagKind uint8

const (
	startTag tagKind = iota
	endTag
	emptyTag
	otherTag
)

// parseTag returns the local name of a tag and its kind.
func parseTag(t []byte) ([]byte, tagKind) {
	if len(t) == 0 {
		return nil, otherTag
	}
	kind := startTag
	switch t[0] {
	case '/':
		kind = endTag
		t = t[1:]
	case '?', '!':
		return nil, otherTag
	}
	if len(t) > 0 && t[len(t)-1] == '/' {
		kind = emptyTag
		t = t[:len(t)-1]
	}
	end := 0
	for end < len(t) && !isSpace(t[end]) {
		end++
	}
	name := t[:end]
	if i := bytes.IndexByte(name, ':'); i >= 0 {
		name = name[i+1:]
	}
	return name, kind
}

// cdata returns the content of a CDATA tag.
func cdata(t []byte) ([]byte, bool) {
	if bytes.HasPrefix(t, []byte("![CDATA[")) && bytes.HasSuffix(t, []byte("]]")) {
		return t[8 : len(t)-2], true
	}
	return nil, false
}

// attrs calls fn for each attribute of a start tag with its raw value.
func attrs(t []byte, fn func(name, value []byte)) {
	i := 0
	for i < len(t) && !isSpace(t[i]) {
		i++
	}
	for i < len(t) {
		for i < len(t) && isSpace(t[i]) {
			i++
		}
		start := i
		for i < len(t) && t[i] != '=' && !isSpace(t[i]) {
			i++
		}
		name := t[start:i]
		for i < len(t) && (t[i] == '=' || isSpace(t[i])) {
			i++
		}
		if i >= len(t) || (t[i] != '"' && t[i] != '\'') {
			return
		}
		q := t[i]
		end := bytes.IndexByte(t[i+1:], q)
		if end < 0 {
			return
		}
		if j := bytes.IndexByte(name, ':'); j >= 0 {
			name = name[j+1:]
		}
		fn(name, t[i+1:i+1+end])
		i += end + 2
	}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// unescape appends text with XML entities decoded and line ends normalized.
func unescape(dst, s []byte) []byte {
	for {
		i := bytes.IndexByte(s, '&')
		if i < 0 {
			return appendLines(dst, s)
		}
		dst = appendLines(dst, s[:i])
		s = s[i:]
		end := bytes.IndexByte(s, ';')
		if end < 0 {
			return append(dst, s...)
		}
		switch ent := s[1:end]; string(ent) {
		case "amp":
			dst = append(dst, '&')
		case "lt":
			dst = append(dst, '<')
		case "gt":
			dst = append(dst, '>')
		case "quot":
			dst = append(dst, '"')
		case "apos":
			dst = append(dst, '\'')
		default:
			r, ok := charRef(ent)
			if !ok {
				dst = append(dst, s[:end+1]...)
				break
			}
			dst = utf8.AppendRune(dst, r)
		}
		s = s[end+1:]
	}
}

// appendLines appends s with \r\n and lone \r turned into \n, as XML requires.
func appendLines(dst, s []byte) []byte {
	for {
		i := bytes.IndexByte(s, '\r')
		if i < 0 {
			return append(dst, s...)
		}
		dst = append(append(dst, s[:i]...), '\n')
		s = s[i+1:]
		if len(s) > 0 && s[0] == '\n' {
			s = s[1:]
		}
	}
}

func charRef(ent []byte) (rune, bool) {
	if len(ent) < 2 || ent[0] != '#' {
		return 0, false
	}
	base, digits := 10, ent[1:]
	if digits[0] == 'x' || digits[0] == 'X' {
		base, digits = 16, digits[1:]
	}
	n, err := strconv.ParseUint(string(digits), base, 32)
	if err != nil || !utf8.ValidRune(rune(n)) {
		return 0, false
	}
	return rune(n), true
}

// atoi parses a small non-negative decimal number, -1 if invalid.
func atoi(b []byte) int {
	if len(b) == 0 || len(b) > 9 {
		return -1
	}
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}
