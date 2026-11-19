// Command grid prints CSV, xlsx and xls files as tables.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"golang.org/x/term"

	"gridview/internal/output"
	"gridview/internal/reader"
	"gridview/internal/table"
	"gridview/internal/tui"
)

func main() {
	err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "grid:", err)
		os.Exit(1)
	}
}

type config struct {
	path  string
	view  bool
	sheet string
	delim rune
	width int
	yes   bool
	skip  int
	cols  string
	// noHeader treats the first row as data.
	noHeader bool
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cfg, err := parseArgs(args, stderr)
	if err != nil {
		return err
	}
	rows, err := open(cfg, stdin)
	if err != nil {
		return err
	}
	defer rows.Close()
	rows = table.Reshape(rows, cfg.skip, !cfg.noHeader)
	if cfg.cols != "" {
		if rows, err = table.Select(rows, cfg.cols); err != nil {
			return err
		}
	}
	if cfg.view {
		term, restore, err := tui.OpenTerminal()
		if err != nil {
			return err
		}
		defer restore()
		return tui.Run(rows, title(cfg.path), term)
	}
	text := output.Text{W: stdout, MaxWidth: textWidth(cfg.width, stdout), Border: isTerminal(stdout)}
	if !cfg.yes && isTerminal(stdout) {
		text.Confirm = func(n int, more bool) bool {
			return confirm(stderr, n, more)
		}
	}
	return text.Show(rows)
}

// confirm asks before flooding the terminal. Without a console to ask,
// it prints anyway.
func confirm(w io.Writer, n int, more bool) bool {
	count := fmt.Sprint(n)
	if more {
		count += "+"
	}
	ok, err := tui.Ask(w, fmt.Sprintf("Print %s rows? Use -view to browse instead. [y/N] ", count))
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return true
	}
	return ok
}

func title(path string) string {
	if path == "" || path == "-" {
		return "stdin"
	}
	return filepath.Base(path)
}

func open(cfg config, stdin io.Reader) (table.Rows, error) {
	opt := reader.Options{Delimiter: cfg.delim, Sheet: cfg.sheet}
	switch cfg.path {
	case "":
		if isTerminal(stdin) {
			return nil, errors.New("no file given, see grid -h")
		}
		return reader.OpenReader(stdin, opt)
	case "-":
		return reader.OpenReader(stdin, opt)
	}
	return reader.Open(cfg.path, opt)
}

func parseArgs(args []string, stderr io.Writer) (config, error) {
	var cfg config
	var delim string
	fs := flag.NewFlagSet("grid", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&cfg.view, "view", false, "interactive view")
	fs.BoolVar(&cfg.view, "v", false, "short for -view")
	fs.StringVar(&cfg.sheet, "sheet", "", "sheet name or 1-based index (xlsx, xls)")
	fs.StringVar(&cfg.sheet, "s", "", "short for -sheet")
	fs.StringVar(&delim, "d", "", "CSV delimiter, default auto (use tab for tabs)")
	fs.IntVar(&cfg.width, "width", -1, "max line width, 0 unlimited, default terminal width")
	fs.IntVar(&cfg.width, "w", -1, "short for -width")
	fs.BoolVar(&cfg.yes, "yes", false, "print long tables without asking")
	fs.BoolVar(&cfg.yes, "y", false, "short for -yes")
	fs.BoolVar(&cfg.noHeader, "noheader", false, "the first row is data, name the columns A, B, C")
	fs.StringVar(&cfg.cols, "cols", "", "columns to show, by name, number or range: name,city or 1,3-5")
	fs.StringVar(&cfg.cols, "c", "", "short for -cols")
	fs.IntVar(&cfg.skip, "skip", 0, "skip this many rows at the top, before the header")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: grid [flags] [file]\n\nfile is a .csv, .xlsx or .xls file, - or nothing reads stdin\n\nflags:")
		fs.PrintDefaults()
	}
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return cfg, err
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
	if len(pos) > 1 {
		return cfg, fmt.Errorf("expected one file, got %d", len(pos))
	}
	if len(pos) == 1 {
		cfg.path = pos[0]
	}
	if cfg.skip < 0 {
		return cfg, fmt.Errorf("invalid -skip %d", cfg.skip)
	}
	d, err := parseDelim(delim)
	cfg.delim = d
	return cfg, err
}

func parseDelim(s string) (rune, error) {
	switch s {
	case "":
		return 0, nil
	case "tab", `\t`:
		return '\t', nil
	}
	r, n := utf8.DecodeRuneInString(s)
	if n != len(s) || r == '"' || r == '\n' || r == '\r' || r == utf8.RuneError {
		return 0, fmt.Errorf("invalid delimiter %q", s)
	}
	return r, nil
}

func textWidth(flagWidth int, w io.Writer) int {
	if flagWidth >= 0 {
		return flagWidth
	}
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return 0
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0
	}
	return width
}

func isTerminal(v any) bool {
	f, ok := v.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
