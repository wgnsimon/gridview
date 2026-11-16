package table

import "io"

type sliceRows struct {
	header []string
	rows   [][]string
	next   int
}

// FromSlices returns Rows over in-memory data.
func FromSlices(header []string, rows [][]string) Rows {
	return &sliceRows{header: header, rows: rows}
}

func (s *sliceRows) Header() []string {
	return s.header
}

func (s *sliceRows) Next() ([]string, error) {
	if s.next >= len(s.rows) {
		return nil, io.EOF
	}
	s.next++
	return s.rows[s.next-1], nil
}

func (s *sliceRows) Close() error {
	return nil
}
