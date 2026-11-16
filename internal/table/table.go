// Package table defines the row stream shared by all readers and outputs.
package table

// Rows streams a table row by row. Next returns io.EOF at the end.
// A slice returned by Next is only valid until the next call.
type Rows interface {
	Header() []string
	Next() ([]string, error)
	Close() error
}
