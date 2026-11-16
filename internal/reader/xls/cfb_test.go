package xls

import (
	"bytes"
	"os"
	"testing"
)

// TestCFBMiniStream reads a stream small enough to live in the mini stream.
func TestCFBMiniStream(t *testing.T) {
	data, err := os.ReadFile("../testdata/libreoffice.xls")
	if err != nil {
		t.Fatal(err)
	}
	c, err := openCFB(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	info, _, err := c.stream("\x05SummaryInformation")
	if err != nil {
		t.Fatal(err)
	}
	if len(info) >= int(c.cutoff) || !bytes.HasPrefix(info, []byte{0xFE, 0xFF}) {
		t.Errorf("SummaryInformation: %d bytes, starts % x", len(info), info[:min(4, len(info))])
	}
}
