package xlsx

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"
)

func TestPrefetch(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 1000)
	for _, size := range []int{1, 7, 4096, 20000} {
		p := newPrefetch(io.NopCloser(iotest.HalfReader(bytes.NewReader(data))), size, 2)
		got, err := io.ReadAll(p)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("size %d: got %d bytes", size, len(got))
		}
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPrefetchError(t *testing.T) {
	boom := errors.New("boom")
	p := newPrefetch(io.NopCloser(iotest.ErrReader(boom)), 16, 2)
	if _, err := io.ReadAll(p); err != boom {
		t.Errorf("err = %v", err)
	}
	p.Close()
}

func TestPrefetchCloseEarly(t *testing.T) {
	p := newPrefetch(io.NopCloser(bytes.NewReader(make([]byte, 1<<20))), 1024, 2)
	buf := make([]byte, 10)
	if _, err := p.Read(buf); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
