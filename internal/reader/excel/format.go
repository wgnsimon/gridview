// Package excel holds what xlsx and xls share: number formats, dates and sheet lookup.
package excel

import (
	"math"
	"strconv"
	"strings"
	"time"
)

type Kind uint8

const (
	General Kind = iota
	Fixed
	Percent
	Date
	Time
	DateTime
	Text
)

// Format describes how a number is shown.
type Format struct {
	Kind     Kind
	Decimals int
}

// DateSystem is the epoch of date serial numbers.
type DateSystem uint8

const (
	Date1900 DateSystem = iota // Excel default, keeps the 1900 leap year bug
	Date1904
	Date1899 // LibreOffice without date compatibility
)

// Builtin returns the format of a built-in number format id.
func Builtin(id int) Format {
	switch id {
	case 1, 3, 37, 38:
		return Format{Kind: Fixed}
	case 2, 4, 39, 40:
		return Format{Kind: Fixed, Decimals: 2}
	case 9:
		return Format{Kind: Percent}
	case 10:
		return Format{Kind: Percent, Decimals: 2}
	case 14, 15, 16, 17, 27, 28, 29, 30, 31, 34, 35, 36, 50, 51, 52, 53, 54, 57, 58:
		return Format{Kind: Date}
	case 18, 19, 20, 21, 32, 33, 45, 46, 47, 55, 56:
		return Format{Kind: Time}
	case 22:
		return Format{Kind: DateTime}
	case 49:
		return Format{Kind: Text}
	}
	return Format{}
}

// ParseFormat classifies a custom number format code like "dd.mm.yyyy" or "0.00%".
func ParseFormat(code string) Format {
	code = strings.ToLower(firstSection(code))
	switch code {
	case "", "general":
		return Format{}
	case "@":
		return Format{Kind: Text}
	}
	var date, clock, percent, digits, exp bool
	decimals, afterPoint := 0, false
	for i := 0; i < len(code); i++ {
		switch c := code[i]; c {
		case 'y', 'd':
			date = true
		case 'm':
			if minutes(code, i) {
				clock = true
			} else if !clock {
				date = true
			}
		case 'h', 's':
			clock = true
		case 'a':
			if strings.HasPrefix(code[i:], "am/pm") || strings.HasPrefix(code[i:], "a/p") {
				clock = true
			}
		case 'e':
			exp = true
		case '%':
			percent = true
		case '.':
			afterPoint = true
		case '0', '#', '?':
			digits = true
			if afterPoint {
				decimals++
			}
		}
	}
	switch {
	case date && clock:
		return Format{Kind: DateTime}
	case clock:
		return Format{Kind: Time}
	case date:
		return Format{Kind: Date}
	case exp || !digits:
		return Format{}
	case percent:
		return Format{Kind: Percent, Decimals: decimals}
	}
	return Format{Kind: Fixed, Decimals: decimals}
}

// minutes reports whether the m at i means minutes, as in "mm:ss".
func minutes(code string, i int) bool {
	for i < len(code) && code[i] == 'm' {
		i++
	}
	for i < len(code) && (code[i] == ':' || code[i] == ' ') {
		i++
	}
	return i < len(code) && code[i] == 's'
}

// firstSection returns the positive section of a format code without
// quoted text, escapes and bracketed parts like colors or locales.
func firstSection(code string) string {
	var b strings.Builder
	for i := 0; i < len(code); i++ {
		switch c := code[i]; c {
		case ';':
			return b.String()
		case '"':
			for i++; i < len(code) && code[i] != '"'; i++ {
			}
		case '\\', '_', '*':
			i++
		case '[':
			end := strings.IndexByte(code[i:], ']')
			if end < 0 {
				return b.String()
			}
			if inner := strings.ToLower(code[i+1 : i+end]); inner == "h" || inner == "hh" || inner == "m" || inner == "mm" || inner == "s" || inner == "ss" {
				b.WriteString(inner)
			}
			i += end
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// FormatNumber renders v as text according to f.
func FormatNumber(v float64, f Format, ds DateSystem) string {
	switch f.Kind {
	case Fixed:
		return strconv.FormatFloat(v, 'f', f.Decimals, 64)
	case Percent:
		return strconv.FormatFloat(v*100, 'f', f.Decimals, 64) + "%"
	case Date:
		return datePart(v, ds)
	case Time:
		return clockPart(v)
	case DateTime:
		return datePart(v, ds) + " " + clockPart(v)
	}
	return GeneralNumber(v)
}

// GeneralNumber renders v like Excel's General format with 15 significant digits.
func GeneralNumber(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)
	if a := math.Abs(r); a >= 1e15 || (a != 0 && a < 1e-9) {
		return strconv.FormatFloat(r, 'g', -1, 64)
	}
	return strconv.FormatFloat(r, 'f', -1, 64)
}

func seconds(v float64) (days, secs int64) {
	total := int64(math.Round(v * 86400))
	return total / 86400, total % 86400
}

func datePart(v float64, ds DateSystem) string {
	days, _ := seconds(v)
	var epoch time.Time
	switch {
	case ds == Date1904:
		epoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	case ds == Date1900 && days == 60:
		return "1900-02-29"
	case ds == Date1900 && days < 60:
		epoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)
	default:
		epoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	}
	return epoch.AddDate(0, 0, int(days)).Format("2006-01-02")
}

func clockPart(v float64) string {
	_, secs := seconds(v)
	if secs < 0 {
		secs += 86400
	}
	b := make([]byte, 0, 8)
	b = appendTwo(b, secs/3600)
	b = append(b, ':')
	b = appendTwo(b, secs/60%60)
	b = append(b, ':')
	return string(appendTwo(b, secs%60))
}

func appendTwo(b []byte, n int64) []byte {
	return append(b, byte('0'+n/10), byte('0'+n%10))
}
