// Package store keeps captured requests in memory, in arrival order, within
// a fixed budget.
package store

import (
	"context"
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
	requests       int
	events         int
	control        int
	evicted        int
	// changed is closed on the next Capture. It is made only when a waiter
	// asks for it, so a store without waiters allocates nothing per request.
	changed chan struct{}

	// onWait, when set, runs each time a waiter blocks. Tests use it to
	// append only after a waiter has scanned.
	onWait func()
}

// Stats describes what the store holds now. Requests counts ingestion
// requests, whatever their answer; Control counts the other captured requests.
type Stats struct {
	Requests       int    `json:"requests"`
	Events         int    `json:"events"`
	Control        int    `json:"control"`
	Bytes          int    `json:"bytes"`
	Evicted        int    `json:"evicted"`
	EvictedThrough uint64 `json:"evictedThrough"`
	MaxRequests    int    `json:"maxRequests"`
	MaxBytes       int    `json:"maxBytes"`
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
	s.count(rec, 1)
	for len(s.records) > s.maxRequests || s.bytes > s.maxBytes {
		oldest := s.records[0]
		s.records[0] = nil
		s.records = s.records[1:]
		s.count(oldest, -1)
		s.evicted++
		s.evictedThrough = oldest.Seq
	}
	if s.changed != nil {
		close(s.changed)
		s.changed = nil
	}
}

// count adds rec to the totals, or removes it when sign is -1.
func (s *Store) count(rec *Record, sign int) {
	s.bytes += sign * rec.size
	s.events += sign * len(rec.Events)
	if rec.Kind == "control" {
		s.control += sign
	} else {
		s.requests += sign
	}
}

// Stats returns the totals over the records the store holds.
func (s *Store) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Stats{
		Requests:       s.requests,
		Events:         s.events,
		Control:        s.control,
		Bytes:          s.bytes,
		Evicted:        s.evicted,
		EvictedThrough: s.evictedThrough,
		MaxRequests:    s.maxRequests,
		MaxBytes:       s.maxBytes,
	}
}

// Cursor returns the highest seq stored so far, 0 before the first request.
func (s *Store) Cursor() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seq
}

// Since returns the records after seq, oldest first, and the seq of the
// newest evicted record. Records never change once stored.
func (s *Store) Since(seq uint64) ([]*Record, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.after(seq), s.evictedThrough
}

// after needs s.mu. It returns a copy, because eviction clears the slots of
// the shared array.
func (s *Store) after(seq uint64) []*Record {
	i := sort.Search(len(s.records), func(i int) bool { return s.records[i].Seq > seq })
	return slices.Clone(s.records[i:])
}

// Wait sums count over the records after since until the sum reaches
// atLeast, then returns it. The sum counts arrivals, so a record that
// eviction removed during the wait still counts. It returns the sum so far and ctx.Err() when ctx ends
// first. Each wake scans only the records stored since the last scan, so
// many waiters on a full store cost little per request.
func (s *Store) Wait(ctx context.Context, since uint64, atLeast int, count func(*Record) int) (int, error) {
	found := 0
	for {
		s.mu.Lock()
		records := s.after(since)
		// Taken under the same lock as the scan, so no Capture falls between.
		if s.changed == nil {
			s.changed = make(chan struct{})
		}
		changed := s.changed
		s.mu.Unlock()

		for _, r := range records {
			found += count(r)
			since = r.Seq
		}
		if found >= atLeast {
			return found, nil
		}
		if s.onWait != nil {
			s.onWait()
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return found, ctx.Err()
		}
	}
}
