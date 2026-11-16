package tui

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gridview/internal/reader"
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
	term := &eofTerm{}
	if err := Run(table.FromSlices([]string{"a"}, nil), "x", term); err != nil {
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

var update = flag.Bool("update", false, "rewrite the golden files")

// TestSession drives the view over a real workbook and compares the final screen.
func TestSession(t *testing.T) {
	rows, err := reader.Open("../reader/testdata/types.xlsx", reader.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	term := &fakeTerm{steps: []step{{"row 1/5", "jjl/end\r"}, {"row 5/5", "q"}}}
	if err := Run(rows, "types.xlsx", term); err != nil {
		t.Fatal(err)
	}
	frames := strings.Split(term.output(), "\x1b[?2026h")
	got := ansi.ReplaceAllString(frames[len(frames)-1], "")
	got = strings.ReplaceAll(got, "\r\n", "\n")
	path := filepath.Join("testdata", "session.golden")
	if *update {
		os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("screen differs from %s:\n%s", path, got)
	}
}
