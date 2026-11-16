package tui

import (
	"io"
	"sync"
	"time"

	"gridview/internal/layout"
	"gridview/internal/table"
)

// Terminal is what the view needs: key input, output and the window size.
type Terminal interface {
	io.Reader
	io.Writer
	Size() (width, height int, err error)
}

// Run shows rows until the user quits. Rows load in the background.
func Run(rows table.Rows, title string, term Terminal) error {
	w, h, err := term.Size()
	if err != nil {
		return err
	}
	m := &model{title: title, header: rows.Header(), store: &table.Store{}, loading: true, width: w, height: h}
	l := startLoader(rows, m.store, m.header)
	defer l.stop()

	keys := make(chan []Key)
	go readKeys(term, keys)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	if _, err := io.WriteString(term, "\x1b[?1049h\x1b[?25l"); err != nil {
		return err
	}
	defer io.WriteString(term, "\x1b[?25h\x1b[?1049l")

	frame := make([]byte, 0, 64<<10)
	dirty := true
	for {
		if dirty {
			frame = m.render(frame[:0])
			if _, err := term.Write(frame); err != nil {
				return err
			}
			dirty = false
		}
		select {
		case ks, ok := <-keys:
			if !ok {
				return nil
			}
			for _, k := range ks {
				if m.handle(k) {
					return nil
				}
			}
			dirty = true
		case <-l.notify:
			l.sync(m)
			dirty = true
		case <-tick.C:
			if w, h, err := term.Size(); err == nil && (w != m.width || h != m.height) {
				m.width, m.height = w, h
				m.clamp()
				dirty = true
			}
		}
	}
}

func readKeys(r io.Reader, out chan<- []Key) {
	buf := make([]byte, 256)
	var pending []byte
	for {
		n, err := r.Read(buf)
		pending = append(pending, buf[:n]...)
		keys, rest := parseKeys(pending)
		pending = append(pending[:0], rest...)
		if len(keys) > 0 {
			out <- keys
		}
		if err != nil {
			close(out)
			return
		}
	}
}

// loader reads rows into the store and publishes progress.
type loader struct {
	mu     sync.Mutex
	rows   int
	cols   []layout.Column
	done   bool
	err    error
	notify chan struct{}
	quit   chan struct{}
	wg     sync.WaitGroup
}

func startLoader(rows table.Rows, store *table.Store, header []string) *loader {
	l := &loader{notify: make(chan struct{}, 1), quit: make(chan struct{})}
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.run(rows, store, header)
	}()
	return l
}

func (l *loader) run(rows table.Rows, store *table.Store, header []string) {
	var ms layout.Measure
	ms.AddHeader(header)
	last := time.Now()
	for n := 1; ; n++ {
		row, err := rows.Next()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			l.publish(n-1, &ms, true, err)
			return
		}
		store.Append(row)
		ms.Add(row)
		if n == 100 || n&4095 == 0 {
			select {
			case <-l.quit:
				return
			default:
			}
			if n == 100 || time.Since(last) > 50*time.Millisecond {
				l.publish(n, &ms, false, nil)
				last = time.Now()
			}
		}
	}
}

func (l *loader) publish(n int, ms *layout.Measure, done bool, err error) {
	cols := ms.Columns()
	l.mu.Lock()
	l.rows, l.cols, l.done, l.err = n, cols, done, err
	l.mu.Unlock()
	select {
	case l.notify <- struct{}{}:
	default:
	}
}

func (l *loader) sync(m *model) {
	l.mu.Lock()
	m.rows, m.cols, m.loading, m.err = l.rows, l.cols, !l.done, l.err
	l.mu.Unlock()
	m.clamp()
}

// stop ends loading; a reader blocked on slow input is left behind.
func (l *loader) stop() {
	close(l.quit)
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
	}
}
