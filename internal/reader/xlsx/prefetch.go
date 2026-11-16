package xlsx

import (
	"io"
	"sync"
)

// prefetch inflates the next chunks in the background while the current one is parsed.
type prefetch struct {
	rc   io.ReadCloser
	full chan chunk
	free chan []byte
	stop chan struct{}
	wg   sync.WaitGroup
	cur  []byte
	buf  []byte
	err  error
}

type chunk struct {
	data []byte
	err  error
}

func newPrefetch(rc io.ReadCloser, size, n int) *prefetch {
	p := &prefetch{rc: rc, full: make(chan chunk, n), free: make(chan []byte, n), stop: make(chan struct{})}
	for range n {
		p.free <- make([]byte, size)
	}
	p.wg.Add(1)
	go p.fill()
	return p
}

func (p *prefetch) fill() {
	defer p.wg.Done()
	for {
		var buf []byte
		select {
		case buf = <-p.free:
		case <-p.stop:
			return
		}
		n, err := io.ReadFull(p.rc, buf)
		if err == io.ErrUnexpectedEOF {
			err = io.EOF
		}
		select {
		case p.full <- chunk{buf[:n], err}:
		case <-p.stop:
			return
		}
		if err != nil {
			return
		}
	}
}

func (p *prefetch) Read(b []byte) (int, error) {
	for len(p.cur) == 0 {
		if p.buf != nil {
			p.free <- p.buf[:cap(p.buf)]
			p.buf = nil
		}
		if p.err != nil {
			return 0, p.err
		}
		c := <-p.full
		p.cur, p.buf, p.err = c.data, c.data, c.err
	}
	n := copy(b, p.cur)
	p.cur = p.cur[n:]
	return n, nil
}

func (p *prefetch) Close() error {
	close(p.stop)
	p.wg.Wait()
	return p.rc.Close()
}
