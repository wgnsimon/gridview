package reader

import (
	"encoding/csv"
	"io"
	"slices"

	"gridview/internal/table"
)

type csvRows struct {
	r      *csv.Reader
	header []string
}

func newCSV(r io.Reader, delim rune) (*csvRows, error) {
	cr := csv.NewReader(r)
	cr.Comma = delim
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.ReuseRecord = true
	header, err := cr.Read()
	if err != nil && err != io.EOF {
		return nil, err
	}
	return &csvRows{r: cr, header: slices.Clone(header)}, nil
}

func (c *csvRows) Header() []string {
	return c.header
}

func (c *csvRows) Next() ([]string, error) {
	return c.r.Read()
}

func (c *csvRows) Close() error {
	return nil
}

var _ table.Rows = (*csvRows)(nil)
