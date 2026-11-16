package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Ask writes question to out and reads a line from the console, even when
// stdin carries the data. Only y or yes count as yes.
func Ask(out io.Writer, question string) (bool, error) {
	in := os.Stdin
	if !term.IsTerminal(int(in.Fd())) {
		tty, err := openTTY()
		if err != nil {
			return false, err
		}
		defer tty.Close()
		in = tty
	}
	fmt.Fprint(out, question)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}
