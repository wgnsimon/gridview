package table

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestStore(t *testing.T) {
	var s Store
	rows := [][]string{
		{"a", "", "äöü"},
		{},
		{strings.Repeat("x", chunkSize+10)},
		{"after big"},
	}
	for _, r := range rows {
		s.Append(r)
	}
	if s.Len() != len(rows) {
		t.Fatalf("Len = %d", s.Len())
	}
	var dst []string
	for i, want := range rows {
		dst = s.Row(i, dst)
		if !slices.Equal(dst, want) {
			t.Errorf("row %d = %.20q, want %.20q", i, dst, want)
		}
	}
}

func TestStoreConcurrent(t *testing.T) {
	var s Store
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 20000 {
			s.Append([]string{fmt.Sprint(i), "v"})
		}
	}()
	var dst []string
	for s.Len() < 20000 {
		if n := s.Len(); n > 0 {
			dst = s.Row(n-1, dst)
			if dst[0] != fmt.Sprint(n-1) {
				t.Fatalf("row %d = %q", n-1, dst)
			}
		}
	}
	wg.Wait()
}
