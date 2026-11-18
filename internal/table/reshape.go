package table

import (
	"io"
	"slices"
)

// Reshape drops the first skip rows of a table, counting its header as the
// first row. The next row becomes the header; if header is false it stays
// data and the columns are named A, B, C like in a spreadsheet.
func Reshape(rows Rows, skip int, header bool) Rows {
	if skip <= 0 && header {
		return rows
	}
	r := &reshaped{Rows: rows}
	row := rows.Header()
	for range skip {
		if row == nil {
			break
		}
		row, r.err = rows.Next()
		if r.err == io.EOF {
			row, r.err = nil, nil
			r.done = true
			break
		}
		if r.err != nil {
			return r
		}
		// A blank spreadsheet row is nil but still a row.
		if row == nil {
			row = []string{}
		}
	}
	if row == nil || r.done {
		r.done = true
		return r
	}
	if header {
		r.header = slices.Clone(row)
		return r
	}
	r.header = make([]string, len(row))
	for i := range r.header {
		r.header[i] = ColumnName(i)
	}
	r.first = slices.Clone(row)
	r.pending = true
	return r
}

type reshaped struct {
	Rows
	header  []string
	first   []string
	pending bool
	done    bool
	err     error
}

func (r *reshaped) Header() []string {
	return r.header
}

func (r *reshaped) Next() ([]string, error) {
	switch {
	case r.err != nil:
		return nil, r.err
	case r.done:
		return nil, io.EOF
	case r.pending:
		r.pending = false
		return r.first, nil
	}
	return r.Rows.Next()
}

// ColumnName returns the spreadsheet name of column i, counted from 0:
// A … Z, AA … ZZ, AAA and so on.
func ColumnName(i int) string {
	var b []byte
	for i++; i > 0; i = (i - 1) / 26 {
		b = append(b, byte('A'+(i-1)%26))
	}
	slices.Reverse(b)
	return string(b)
}
