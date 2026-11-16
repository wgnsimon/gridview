package xls

import (
	"bytes"
	"os"
	"testing"
)

func TestCFBStreams(t *testing.T) {
	for _, file := range []string{"types.xls", "libreoffice.xls"} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile("../testdata/" + file)
			if err != nil {
				t.Fatal(err)
			}
			c, err := openCFB(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			wb, name, err := c.stream("Workbook", "Book")
			if err != nil {
				t.Fatal(err)
			}
			if name != "Workbook" || len(wb) < 1000 || wb[0] != 0x09 || wb[1] != 0x08 {
				t.Errorf("Workbook stream: name %q, %d bytes, starts % x", name, len(wb), wb[:4])
			}
			info, _, err := c.stream("\x05SummaryInformation")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(info, []byte{0xFE, 0xFF}) {
				t.Errorf("SummaryInformation starts % x", info[:min(4, len(info))])
			}
			if file == "libreoffice.xls" && len(info) >= int(c.cutoff) {
				t.Errorf("want a mini stream, got %d bytes", len(info))
			}
			if _, _, err := c.stream("missing"); err == nil {
				t.Error("missing stream: want error")
			}
		})
	}
}

func TestCFBCorrupt(t *testing.T) {
	data, err := os.ReadFile("../testdata/types.xls")
	if err != nil {
		t.Fatal(err)
	}
	for _, cut := range []int{0, 100, 512, 600, len(data) / 2} {
		c, err := openCFB(bytes.NewReader(data[:cut]), int64(cut))
		if err == nil {
			_, _, err = c.stream("Workbook")
		}
		if err == nil {
			t.Errorf("cut at %d: want error", cut)
		}
	}
}
