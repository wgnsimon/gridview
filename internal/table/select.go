package table

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Select keeps the columns named in spec, in that order. spec is a comma
// separated list of header names (ignoring case), numbers counted from 1,
// and ranges such as 2-5 or 3- for the third to the last column.
func Select(rows Rows, spec string) (Rows, error) {
	header := rows.Header()
	var idx []int
	for part := range strings.SplitSeq(spec, ",") {
		part = strings.TrimSpace(part)
		cols, err := resolve(header, part)
		if err != nil {
			return nil, err
		}
		idx = append(idx, cols...)
	}
	s := &selected{Rows: rows, idx: idx}
	s.header = slices.Clone(s.pick(header))
	return s, nil
}

func resolve(header []string, part string) ([]int, error) {
	if part == "" {
		return nil, fmt.Errorf("empty column name in -cols")
	}
	for i, h := range header {
		if strings.EqualFold(strings.TrimSpace(h), part) {
			return []int{i}, nil
		}
	}
	from, to, isRange := strings.Cut(part, "-")
	first, err := strconv.Atoi(from)
	if err != nil {
		return nil, fmt.Errorf("no column %q, columns are: %s", part, strings.Join(header, ", "))
	}
	last := first
	if isRange {
		last = len(header)
		if to != "" {
			if last, err = strconv.Atoi(to); err != nil {
				return nil, fmt.Errorf("invalid range %q", part)
			}
		}
		if last < first {
			return nil, fmt.Errorf("invalid range %q", part)
		}
	}
	for _, n := range []int{first, last} {
		if n < 1 || n > len(header) {
			return nil, fmt.Errorf("no column %d, there are %d", n, len(header))
		}
	}
	idx := make([]int, 0, last-first+1)
	for i := first; i <= last; i++ {
		idx = append(idx, i-1)
	}
	return idx, nil
}

type selected struct {
	Rows
	idx    []int
	header []string
	buf    []string
}

func (s *selected) Header() []string {
	return s.header
}

func (s *selected) Next() ([]string, error) {
	row, err := s.Rows.Next()
	if err != nil {
		return nil, err
	}
	return s.pick(row), nil
}

// pick returns the selected cells of row in a buffer that Next reuses.
func (s *selected) pick(row []string) []string {
	s.buf = s.buf[:0]
	for _, i := range s.idx {
		c := ""
		if i < len(row) {
			c = row[i]
		}
		s.buf = append(s.buf, c)
	}
	return s.buf
}
