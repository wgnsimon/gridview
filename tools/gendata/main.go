// Command gendata writes deterministic order data as CSV or xlsx.
//
//	go run ./tools/gendata -rows 1000000 input/large.csv
//	go run ./tools/gendata -rows 100000 input/medium.xlsx
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gridview/internal/sample"
)

func main() {
	rows := flag.Int("rows", 10000, "number of data rows")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gendata [-rows n] file.csv|file.xlsx")
		os.Exit(2)
	}
	if err := write(flag.Arg(0), *rows); err != nil {
		fmt.Fprintln(os.Stderr, "gendata:", err)
		os.Exit(1)
	}
}

func write(path string, rows int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if strings.EqualFold(filepath.Ext(path), ".xlsx") {
		err = sample.WriteXLSX(f, rows)
	} else {
		err = sample.WriteCSV(f, rows)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
