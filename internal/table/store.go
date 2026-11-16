package table

import (
	"encoding/binary"
	"sync"
	"unsafe"
)

const chunkSize = 1 << 20

// Store keeps rows compactly in large chunks for random access.
// One writer may append while readers read; strings returned by Row stay valid.
type Store struct {
	mu     sync.RWMutex
	chunks [][]byte
	index  []uint64
	buf    []byte
}

// Append copies row into the store.
func (s *Store) Append(row []string) {
	s.buf = binary.AppendUvarint(s.buf[:0], uint64(len(row)))
	for _, c := range row {
		s.buf = binary.AppendUvarint(s.buf, uint64(len(c)))
		s.buf = append(s.buf, c...)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.chunks)
	if n == 0 || cap(s.chunks[n-1])-len(s.chunks[n-1]) < len(s.buf) {
		s.chunks = append(s.chunks, make([]byte, 0, max(chunkSize, len(s.buf))))
		n++
	}
	c := s.chunks[n-1]
	s.index = append(s.index, uint64(n-1)<<32|uint64(len(c)))
	s.chunks[n-1] = append(c, s.buf...)
}

// Len returns the number of rows.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index)
}

// Raw returns row i as stored: cell bytes with length prefixes in between.
// It is meant for searching; a hit has to be confirmed on the decoded row.
func (s *Store) Raw(i int) string {
	s.mu.RLock()
	at := s.index[i]
	c := s.chunks[at>>32]
	end := len(c)
	if i+1 < len(s.index) && s.index[i+1]>>32 == at>>32 {
		end = int(uint32(s.index[i+1]))
	}
	s.mu.RUnlock()
	b := c[uint32(at):end]
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Row decodes row i into dst and returns it.
func (s *Store) Row(i int, dst []string) []string {
	s.mu.RLock()
	at := s.index[i]
	c := s.chunks[at>>32]
	s.mu.RUnlock()
	b := c[uint32(at):len(c)]
	n, k := binary.Uvarint(b)
	b = b[k:]
	dst = dst[:0]
	for range n {
		l, k := binary.Uvarint(b)
		b = b[k:]
		if l == 0 {
			dst = append(dst, "")
			continue
		}
		dst = append(dst, unsafe.String(&b[0], int(l)))
		b = b[l:]
	}
	return dst
}
