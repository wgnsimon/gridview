package reader

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"testing/iotest"
	"unicode/utf16"
)

func TestUTF8Text(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"plain", []byte("a,b\nä,ö\n"), "a,b\nä,ö\n"},
		{"bom", append([]byte{0xEF, 0xBB, 0xBF}, "a,ü"...), "a,ü"},
		{"utf16le", encodeUTF16(binary.LittleEndian, "a\tÄ😀\n"), "a\tÄ😀\n"},
		{"utf16be", encodeUTF16(binary.BigEndian, "x€"), "x€"},
		{"utf16 lone surrogate", append([]byte{0xFF, 0xFE}, 0x3D, 0xD8, 'a', 0), "�a"},
		{"cp1252", []byte("M\xfcller;5\x80;\x84x\x93"), "Müller;5€;„x“"},
		{"utf8 cut at peek end", []byte("ab"), "ab"},
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

func encodeUTF16(order binary.ByteOrder, s string) []byte {
	b := []byte{0xFF, 0xFE}
	if order == binary.BigEndian {
		b = []byte{0xFE, 0xFF}
	}
	for _, u := range utf16.Encode([]rune(s)) {
		b = order.(binary.AppendByteOrder).AppendUint16(b, u)
	}
	return b
}
