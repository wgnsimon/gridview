package tui

import (
	"os"

	"golang.org/x/sys/windows"
)

func openTTY() (*os.File, error) {
	return os.OpenFile("CONIN$", os.O_RDWR, 0)
}

func enableVT(f *os.File) func() {
	h := windows.Handle(f.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return func() {}
	}
	windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING|windows.ENABLE_PROCESSED_OUTPUT)
	return func() { windows.SetConsoleMode(h, mode) }
}
