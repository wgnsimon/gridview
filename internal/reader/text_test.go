package reader

import (
	"bufio"
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

func TestUTF8Text(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"bom", append([]byte{0xEF, 0xBB, 0xBF}, "a,ü"...), "a,ü"},
		{"utf16le", []byte{0xFF, 0xFE, 'a', 0, 0xC4, 0, 0x3D, 0xD8, 0x00, 0xDE}, "aÄ😀"},
		{"utf16be", []byte{0xFE, 0xFF, 0, 'x', 0x20, 0xAC}, "x€"},
		{"utf16 lone surrogate", []byte{0xFF, 0xFE, 0x3D, 0xD8, 'a', 0}, "�a"},
		{"cp1252", []byte("M\xfcller;5\x80;\x84x\x93"), "Müller;5€;„x“"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := utf8Text(bufio.NewReader(iotest.OneByteReader(bytes.NewReader(tt.in))))
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLooksUTF8(t *testing.T) {
	if !looksUTF8([]byte("abc\xc3")) {
		t.Error("cut rune at the end should still count as UTF-8")
	}
	if looksUTF8([]byte("a\xfcb")) {
		t.Error("latin-1 byte should not count as UTF-8")
	}
}
