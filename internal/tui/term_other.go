//go:build !windows

package tui

import "os"

func openTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

func enableVT(*os.File) func() {
	return func() {}
}
