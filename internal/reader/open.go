// Package reader opens CSV, xlsx and xls files as table.Rows.
package reader

import (
	"io"
	"os"

	"gridview/internal/table"
)

// Options tune how a file is read. Zero values mean auto.
type Options struct {
	Delimiter rune
	Sheet     string
}

// Open opens the file at path.
func Open(path string, opt Options) (table.Rows, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	rows, err := OpenReader(f, opt)
	if err != nil {
		f.Close()
		return nil, err
	}
	return withCloser{rows, f}, nil
}

// OpenReader reads a table from r.
func OpenReader(r io.Reader, opt Options) (table.Rows, error) {
	delim := opt.Delimiter
	if delim == 0 {
		delim = ','
	}
	return newCSV(r, delim)
}

type withCloser struct {
	table.Rows
	c io.Closer
}

func (w withCloser) Close() error {
	err := w.Rows.Close()
	if cerr := w.c.Close(); err == nil {
		err = cerr
	}
	return err
}
