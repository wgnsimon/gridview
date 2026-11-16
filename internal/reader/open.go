// Package reader opens CSV, xlsx and xls files as table.Rows.
package reader

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"

	"gridview/internal/reader/xlsx"
	"gridview/internal/table"
)

// Options tune how a file is read. Zero values mean auto.
type Options struct {
	Delimiter rune
	Sheet     string
}

type format uint8

const (
	formatCSV format = iota
	formatXLSX
	formatXLS
)

var (
	zipMagic = []byte("PK\x03\x04")
	cfbMagic = []byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1")
)

func detect(head []byte) format {
	switch {
	case bytes.HasPrefix(head, zipMagic):
		return formatXLSX
	case bytes.HasPrefix(head, cfbMagic):
		return formatXLS
	}
	return formatCSV
}

// Open opens the file at path and picks the format from its first bytes.
func Open(path string, opt Options) (table.Rows, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	rows, err := openFile(f, opt)
	if err != nil {
		f.Close()
		return nil, err
	}
	return withCloser{rows, f}, nil
}

func openFile(f *os.File, opt Options) (table.Rows, error) {
	head := make([]byte, len(cfbMagic))
	n, _ := f.ReadAt(head, 0)
	if format := detect(head[:n]); format != formatCSV {
		st, err := f.Stat()
		if err != nil {
			return nil, err
		}
		return openWorkbook(format, f, st.Size(), opt.Sheet)
	}
	return openCSV(bufio.NewReaderSize(f, peekSize), opt.Delimiter)
}

// OpenReader reads a table from a stream. Workbooks are buffered in memory.
func OpenReader(r io.Reader, opt Options) (table.Rows, error) {
	br := bufio.NewReaderSize(r, peekSize)
	head, _ := br.Peek(len(cfbMagic))
	if format := detect(head); format != formatCSV {
		data, err := io.ReadAll(br)
		if err != nil {
			return nil, err
		}
		return openWorkbook(format, bytes.NewReader(data), int64(len(data)), opt.Sheet)
	}
	return openCSV(br, opt.Delimiter)
}

func openWorkbook(format format, r io.ReaderAt, size int64, sheet string) (table.Rows, error) {
	if format == formatXLS {
		return nil, errors.New("xls is not supported yet")
	}
	book, err := xlsx.Open(r, size)
	if err != nil {
		return nil, err
	}
	return book.Rows(sheet)
}

func openCSV(br *bufio.Reader, delim rune) (table.Rows, error) {
	text := utf8Text(br)
	if delim == 0 {
		sample, err := text.Peek(peekSize)
		delim = sniffDelimiter(sample, err != nil)
	}
	return newCSV(text, delim)
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
