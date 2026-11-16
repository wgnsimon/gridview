// Package output prints table.Rows as plain text.
package output

import (
	"fmt"
	"io"
	"slices"

	"gridview/internal/layout"
	"gridview/internal/table"
)

const (
	sampleSize = 1000
	flushSize  = 64 << 10
	sep        = " | "
	boxSep     = " │ "
	// AskAbove is the row count above which Show asks Confirm first.
	AskAbove = 250
)

// Text prints an aligned table. Widths come from the first rows,
// later rows stream through without being kept in memory.
type Text struct {
	W io.Writer
	// MaxWidth limits the line width, 0 means unlimited.
	MaxWidth int
	// Border draws the table in a box-drawing frame instead of ASCII lines.
	Border bool
	// Confirm, if set, is asked before printing more than AskAbove rows.
	// rows is the number read so far, more reports that the input goes on.
	// Show prints nothing when it returns false.
	Confirm func(rows int, more bool) bool
}

func (t Text) Show(rows table.Rows) error {
	header := rows.Header()
	var m layout.Measure
	m.AddHeader(header)
	sample, err := readSample(rows, &m)
	if err != nil {
		return err
	}
	if header == nil && len(sample) == 0 {
		return nil
	}
	if t.Confirm != nil && len(sample) > AskAbove && !t.Confirm(len(sample), len(sample) == sampleSize) {
		return nil
	}
	cols := m.Columns()
	f := layout.Formatter{Sep: sep, Cut: t.MaxWidth > 0, Border: t.Border}
	limit := t.MaxWidth
	if t.Border {
		f.Sep = boxSep
		limit -= layout.BorderWidth
	}
	hidden := 0
	if t.MaxWidth > 0 {
		n := layout.Fit(cols, max(limit, 1), layout.Width(f.Sep))
		hidden = len(cols) - n
		cols = cols[:n]
	}
	f.Cols = cols
	buf := make([]byte, 0, flushSize+4096)
	if t.Border {
		buf = append(f.AppendTop(buf), '\n')
	}
	buf = append(f.AppendRow(buf, header), '\n')
	buf = append(f.AppendRule(buf), '\n')
	for _, row := range sample {
		buf = append(f.AppendRow(buf, row), '\n')
	}
	for len(sample) == sampleSize {
		row, err := rows.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		buf = append(f.AppendRow(buf, row), '\n')
		if len(buf) >= flushSize {
			if _, err := t.W.Write(buf); err != nil {
				return err
			}
			buf = buf[:0]
		}
	}
	if t.Border {
		buf = append(f.AppendBottom(buf), '\n')
	}
	if hidden > 0 {
		buf = fmt.Appendf(buf, "(%d more columns, use a wider terminal or -view)\n", hidden)
	}
	_, err = t.W.Write(buf)
	return err
}

func readSample(rows table.Rows, m *layout.Measure) ([][]string, error) {
	var sample [][]string
	for len(sample) < sampleSize {
		row, err := rows.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		row = slices.Clone(row)
		m.Add(row)
		sample = append(sample, row)
	}
	return sample, nil
}
