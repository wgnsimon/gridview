package tui

import (
	"errors"
	"os"

	"golang.org/x/term"
)

type console struct {
	in, out *os.File
}

func (c *console) Read(p []byte) (int, error) {
	return c.in.Read(p)
}

func (c *console) Write(p []byte) (int, error) {
	return c.out.Write(p)
}

func (c *console) Size() (int, int, error) {
	return term.GetSize(int(c.out.Fd()))
}

// OpenTerminal switches the console to raw mode. Keys come from the console
// even when stdin carries the data.
func OpenTerminal() (Terminal, func(), error) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, nil, errors.New("-view needs a terminal on stdout")
	}
	in := os.Stdin
	if !term.IsTerminal(int(in.Fd())) {
		tty, err := openTTY()
		if err != nil {
			return nil, nil, err
		}
		in = tty
	}
	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, nil, err
	}
	restoreOut := enableVT(os.Stdout)
	restore := func() {
		restoreOut()
		term.Restore(int(in.Fd()), state)
		if in != os.Stdin {
			in.Close()
		}
	}
	return &console{in: in, out: os.Stdout}, restore, nil
}
