package reader

import (
	"bufio"
	"bytes"
	"io"
	"slices"
	"strings"

	"gridview/internal/table"
)

// csvRows parses CSV like encoding/csv with LazyQuotes and any number of fields,
// but lines without quotes are split without copying field by field.
type csvRows struct {
	r      *bufio.Reader
	delim  string
	sep    []byte
	header []string
	fields []string
	buf    []byte
	ends   []int
	line   []byte
}

func newCSV(r *bufio.Reader, delim rune) (*csvRows, error) {
	c := &csvRows{r: r, delim: string(delim), sep: []byte(string(delim))}
	header, err := c.Next()
	if err != nil && err != io.EOF {
		return nil, err
	}
	c.header = slices.Clone(header)
	return c, nil
}

func (c *csvRows) Header() []string {
	return c.header
}

func (c *csvRows) Close() error {
	return nil
}

func (c *csvRows) Next() ([]string, error) {
	var line []byte
	for {
		var err error
		line, err = c.readLine()
		if len(line) == 0 && err != nil {
			return nil, err
		}
		if len(line) > lengthNL(line) {
			break
		}
	}
	if bytes.IndexByte(line, '"') < 0 {
		return c.split(string(line[:len(line)-lengthNL(line)])), nil
	}
	return c.quoted(line)
}

// split cuts an unquoted line at every delimiter.
func (c *csvRows) split(s string) []string {
	c.fields = c.fields[:0]
	for {
		var i int
		if len(c.delim) == 1 {
			i = strings.IndexByte(s, c.delim[0])
		} else {
			i = strings.Index(s, c.delim)
		}
		if i < 0 {
			return append(c.fields, s)
		}
		c.fields = append(c.fields, s[:i])
		s = s[i+len(c.delim):]
	}
}

// quoted parses a record with quotes, which may continue on the next lines.
func (c *csvRows) quoted(line []byte) ([]string, error) {
	c.buf, c.ends = c.buf[:0], c.ends[:0]
field:
	for {
		if len(line) == 0 || line[0] != '"' {
			i := c.index(line)
			field := line
			if i >= 0 {
				field = field[:i]
			} else {
				field = field[:len(field)-lengthNL(field)]
			}
			c.buf = append(c.buf, field...)
			c.ends = append(c.ends, len(c.buf))
			if i < 0 {
				return c.finish(), nil
			}
			line = line[i+len(c.delim):]
			continue
		}
		line = line[1:]
		for {
			i := bytes.IndexByte(line, '"')
			switch {
			case i >= 0:
				c.buf = append(c.buf, line[:i]...)
				line = line[i+1:]
				switch {
				case len(line) > 0 && line[0] == '"':
					c.buf = append(c.buf, '"')
					line = line[1:]
				case bytes.HasPrefix(line, c.sep):
					line = line[len(c.delim):]
					c.ends = append(c.ends, len(c.buf))
					continue field
				case lengthNL(line) == len(line):
					c.ends = append(c.ends, len(c.buf))
					return c.finish(), nil
				default:
					c.buf = append(c.buf, '"')
				}
			case len(line) > 0:
				c.buf = append(c.buf, line...)
				var err error
				line, err = c.readLine()
				if err != nil && err != io.EOF {
					return nil, err
				}
			default:
				c.ends = append(c.ends, len(c.buf))
				return c.finish(), nil
			}
		}
	}
}

func (c *csvRows) index(line []byte) int {
	if len(c.sep) == 1 {
		return bytes.IndexByte(line, c.sep[0])
	}
	return bytes.Index(line, c.sep)
}

func (c *csvRows) finish() []string {
	s := string(c.buf)
	c.fields = c.fields[:0]
	start := 0
	for _, end := range c.ends {
		c.fields = append(c.fields, s[start:end])
		start = end
	}
	return c.fields
}

// readLine returns the next line with \r\n turned into \n, like encoding/csv.
func (c *csvRows) readLine() ([]byte, error) {
	line, err := c.r.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		c.line = append(c.line[:0], line...)
		for err == bufio.ErrBufferFull {
			line, err = c.r.ReadSlice('\n')
			c.line = append(c.line, line...)
		}
		line = c.line
	}
	if len(line) > 0 && err == io.EOF {
		err = nil
		if line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
	}
	if n := len(line); n >= 2 && line[n-2] == '\r' && line[n-1] == '\n' {
		line[n-2] = '\n'
		line = line[:n-1]
	}
	return line, err
}

func lengthNL(b []byte) int {
	if len(b) > 0 && b[len(b)-1] == '\n' {
		return 1
	}
	return 0
}

var _ table.Rows = (*csvRows)(nil)
