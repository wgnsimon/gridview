package excel

import "testing"

func TestParseFormat(t *testing.T) {
	tests := []struct {
		code string
		want Format
	}{
		{"General", Format{}},
		{"@", Format{Kind: Text}},
		{"0", Format{Kind: Fixed}},
		{"0.000", Format{Kind: Fixed, Decimals: 3}},
		{"#,##0.00 \"€\"", Format{Kind: Fixed, Decimals: 2}},
		{"#,##0.00;[Red]-#,##0.00", Format{Kind: Fixed, Decimals: 2}},
		{"0.0%", Format{Kind: Percent, Decimals: 1}},
		{"0.00E+00", Format{}},
		{`dd\.mm\.yyyy`, Format{Kind: Date}},
		{"[$-407]mmmm yy", Format{Kind: Date}},
		{"mmm-yy", Format{Kind: Date}},
		{"hh:mm", Format{Kind: Time}},
		{"[h]:mm:ss", Format{Kind: Time}},
		{"mm:ss", Format{Kind: Time}},
		{"mm:ss.0", Format{Kind: Time}},
		{"h:mm AM/PM", Format{Kind: Time}},
		{"yyyy-mm-dd hh:mm:ss", Format{Kind: DateTime}},
		{`"Datum: "dd.mm.yyyy`, Format{Kind: Date}},
		{"[Red]0.00", Format{Kind: Fixed, Decimals: 2}},
		{`"text only"`, Format{}},
	}
	for _, tt := range tests {
		if got := ParseFormat(tt.code); got != tt.want {
			t.Errorf("ParseFormat(%q) = %+v, want %+v", tt.code, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		v    float64
		f    Format
		ds   DateSystem
		want string
	}{
		{42, Format{}, Date1900, "42"},
		{0.30000000000000004, Format{}, Date1900, "0.3"},
		{1234567.8910000001, Format{}, Date1900, "1234567.891"},
		{-3.5, Format{}, Date1900, "-3.5"},
		{1e20, Format{}, Date1900, "1e+20"},
		{1e-12, Format{}, Date1900, "1e-12"},
		{1234567.891, Format{Kind: Fixed, Decimals: 2}, Date1900, "1234567.89"},
		{0.125, Format{Kind: Percent, Decimals: 2}, Date1900, "12.50%"},
		{0.25, Format{Kind: Percent}, Date1900, "25%"},
		{45322, Format{Kind: Date}, Date1900, "2024-01-31"},
		{1, Format{Kind: Date}, Date1900, "1900-01-01"},
		{59, Format{Kind: Date}, Date1900, "1900-02-28"},
		{60, Format{Kind: Date}, Date1900, "1900-02-29"},
		{61, Format{Kind: Date}, Date1900, "1900-03-01"},
		{61, Format{Kind: Date}, Date1899, "1900-03-01"},
		{60, Format{Kind: Date}, Date1899, "1900-02-28"},
		{0, Format{Kind: Date}, Date1904, "1904-01-01"},
		{0.5729166666666666, Format{Kind: Time}, Date1900, "13:45:00"},
		{45322.572916666664, Format{Kind: DateTime}, Date1900, "2024-01-31 13:45:00"},
		{45322.99999999, Format{Kind: DateTime}, Date1900, "2024-02-01 00:00:00"},
	}
	for _, tt := range tests {
		if got := FormatNumber(tt.v, tt.f, tt.ds); got != tt.want {
			t.Errorf("FormatNumber(%v, %+v, %v) = %q, want %q", tt.v, tt.f, tt.ds, got, tt.want)
		}
	}
}
