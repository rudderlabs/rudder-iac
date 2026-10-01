// Package store keeps captured requests in memory, in arrival order, within
// a fixed budget.
package store

import (
	"slices"
	"sort"
	"sync"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
)

const (
	maxRequests = 10_000
	maxBytes    = 64 << 20
)

// Store implements ingest.Sink. It evicts whole requests, oldest first, when
// a new one passes the request or the byte budget.
type Store struct {
	maxRequests int
	maxBytes    int

	mu             sync.Mutex
	records        []*Record
	bytes          int
	seq            uint64
	evictedThrough uint64
}

func New() *Store {
	return &Store{maxRequests: maxRequests, maxBytes: maxBytes}
}

// Capture stores c as the next record. Ingestion limits keep a record far
// below the byte budget, so a new record never evicts itself.
func (s *Store) Capture(c *ingest.Capture) {
	rec := newRecord(c)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	rec.Seq = s.seq
	s.records = append(s.records, rec)
	s.bytes += rec.size
	for len(s.records) > s.maxRequests || s.bytes > s.maxBytes {
		oldest := s.records[0]
		s.records[0] = nil
		s.records = s.records[1:]
		s.bytes -= oldest.size
		s.evictedThrough = oldest.Seq
	}
}

// Since returns the records after seq, oldest first, and the seq of the
// newest evicted record. Records never change once stored.
func (s *Store) Since(seq uint64) ([]*Record, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := sort.Search(len(s.records), func(i int) bool { return s.records[i].Seq > seq })
	return slices.Clone(s.records[i:]), s.evictedThrough
}
