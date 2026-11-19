// Package tui is the interactive table view.
package tui

import "unicode/utf8"

type KeyCode uint8

const (
	KeyRune KeyCode = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyPgUp
	KeyPgDn
	KeyHome
	KeyEnd
	KeyEnter
	KeyEsc
	KeyBackspace
	KeyCtrlC
	KeyCtrlD
	KeyCtrlU
)

// Key is one key press; Rune is set for KeyRune.
type Key struct {
	Code KeyCode
	Rune rune
}

var sequences = map[string]KeyCode{
	"[A": KeyUp, "[B": KeyDown, "[C": KeyRight, "[D": KeyLeft,
	"OA": KeyUp, "OB": KeyDown, "OC": KeyRight, "OD": KeyLeft,
	"[5~": KeyPgUp, "[6~": KeyPgDn,
	"[H": KeyHome, "[1~": KeyHome, "[7~": KeyHome, "OH": KeyHome,
	"[F": KeyEnd, "[4~": KeyEnd, "[8~": KeyEnd, "OF": KeyEnd,
}

// parseKeys turns terminal input into keys and returns an incomplete tail.
func parseKeys(b []byte) ([]Key, []byte) {
	var keys []Key
	for len(b) > 0 {
		c := b[0]
		switch {
		case c == 0x1B:
			if len(b) == 1 || (b[1] != '[' && b[1] != 'O') {
				keys = append(keys, Key{Code: KeyEsc})
				break
			}
			n := sequenceEnd(b)
			if n == 0 {
				return keys, b
			}
			if code, ok := sequences[string(b[1:n])]; ok {
				keys = append(keys, Key{Code: code})
			}
			b = b[n:]
			continue
		case c == 0x03:
			keys = append(keys, Key{Code: KeyCtrlC})
		case c == '\r' || c == '\n':
			keys = append(keys, Key{Code: KeyEnter})
		case c == 0x7F || c == 0x08:
			keys = append(keys, Key{Code: KeyBackspace})
		case c == 0x06:
			keys = append(keys, Key{Code: KeyPgDn})
		case c == 0x02:
			keys = append(keys, Key{Code: KeyPgUp})
		case c == 0x04:
			keys = append(keys, Key{Code: KeyCtrlD})
		case c == 0x15:
			keys = append(keys, Key{Code: KeyCtrlU})
		case c >= utf8.RuneSelf:
			if !utf8.FullRune(b) {
				return keys, b
			}
			r, n := utf8.DecodeRune(b)
			keys = append(keys, Key{Rune: r})
			b = b[n:]
			continue
		case c >= ' ':
			keys = append(keys, Key{Rune: rune(c)})
		}
		b = b[1:]
	}
	return keys, nil
}

// sequenceEnd returns the length of the escape sequence at b, 0 if incomplete.
func sequenceEnd(b []byte) int {
	for i := 2; i < len(b); i++ {
		if c := b[i]; c >= 0x40 && c <= 0x7E {
			return i + 1
		}
	}
	return 0
}
