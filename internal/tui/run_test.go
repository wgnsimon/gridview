package tui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"gridview/internal/table"
)

// fakeTerm types each step's input once the screen shows its text.
type fakeTerm struct {
	mu    sync.Mutex
	out   bytes.Buffer
	steps []step
}

type step struct {
	waitFor, input string
}

func (f *fakeTerm) Read(p []byte) (int, error) {
	if len(f.steps) == 0 {
		select {}
	}
	s := f.steps[0]
	f.steps = f.steps[1:]
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(f.output(), s.waitFor) {
		if time.Now().After(deadline) {
			return 0, errors.New("timeout waiting for " + s.waitFor)
		}
		time.Sleep(time.Millisecond)
	}
	return copy(p, s.input), nil
}

func (f *fakeTerm) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.out.Write(p)
}

func (f *fakeTerm) Size() (int, int, error) {
	return 50, 8, nil
}

func (f *fakeTerm) output() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.out.String()
}

func TestRun(t *testing.T) {
	var rows [][]string
	for i := range 5000 {
		rows = append(rows, []string{fmt.Sprint(i), "x"})
	}
	term := &fakeTerm{steps: []step{{"row 1/5000", "jjG"}, {"row 5000/5000", "q"}}}
	if err := Run(table.FromSlices([]string{"n", "v"}, rows), "data.csv", term); err != nil {
		t.Fatal(err)
	}
	out := term.output()
	if !strings.HasPrefix(out, "\x1b[?1049h") || !strings.HasSuffix(out, "\x1b[?1049l") {
		t.Errorf("alternate screen not entered and left: %q ... %q", out[:20], out[len(out)-20:])
	}
	frames := strings.Split(out, "\x1b[?2026h")
	if last := frames[len(frames)-1]; !strings.Contains(last, "row 5000/5000") || !strings.Contains(last, "4999") {
		t.Errorf("last frame does not show the end:\n%q", last)
	}
}

func TestRunQuitsOnInputEnd(t *testing.T) {
	if err := Run(table.FromSlices([]string{"a"}, nil), "x", &eofTerm{}); err != nil {
		t.Fatal(err)
	}
}

type eofTerm struct {
	bytes.Buffer
}

func (*eofTerm) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (*eofTerm) Size() (int, int, error) {
	return 20, 5, nil
}
