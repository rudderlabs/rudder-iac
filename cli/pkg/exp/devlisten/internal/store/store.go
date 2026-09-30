// Package store keeps captured requests in memory and orders them by seq.
package store

import (
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"
)

const RecordVersion = 1

// Default capacity (contract section 3): the store keeps at most this many
// requests and this many bytes, and evicts the oldest whole requests first.
const (
	DefaultMaxRecords = 10000
	DefaultMaxBytes   = 64 << 20
)

// Record is one captured HTTP request (contract section 3).
type Record struct {
	RecordVersion int        `json:"recordVersion"`
	ServerID      string     `json:"serverId"`
	Seq           uint64     `json:"seq"`
	Kind          string     `json:"kind"`
	Probe         bool       `json:"probe"`
	ReceivedAt    time.Time  `json:"receivedAt"`
	Route         string     `json:"route"`
	Transport     string     `json:"transport"`
	StatusCode    int        `json:"statusCode"`
	Failed        bool       `json:"failed"`
	Outcome       string     `json:"outcome"`
	WriteKey      string     `json:"writeKey"`
	SourceID      string     `json:"sourceId"`
	Rejection     *Rejection `json:"rejection"`
	Hint          *string    `json:"hint"`
	Request       Request    `json:"request"`
	Response      Response   `json:"response"`
	Events        []Event    `json:"events"`
}

type Rejection struct {
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
	Idx    *int   `json:"idx"`
}

type Request struct {
	Method          string      `json:"method"`
	Target          string      `json:"target"`
	Headers         http.Header `json:"headers"`
	RedactedHeaders []string    `json:"redactedHeaders"`
	RemoteAddr      string      `json:"remoteAddr"`
	BodyEncoding    string      `json:"bodyEncoding"`
	BodyBytes       int         `json:"bodyBytes"`
	Body            string      `json:"body"`
	BodyBase64      []byte      `json:"bodyBase64"`
	BodyComplete    bool        `json:"bodyComplete"`
}

type Response struct {
	StatusCode int         `json:"statusCode"`
	Headers    http.Header `json:"headers"`
	Body       string      `json:"body"`
}

// Event is one parsed event of an ingestion request. Scalars are nil when the
// wire value is absent or null.
type Event struct {
	Idx             int             `json:"idx"`
	Type            *string         `json:"type"`
	Event           *string         `json:"event"`
	UserID          *string         `json:"userId"`
	AnonymousID     *string         `json:"anonymousId"`
	MessageID       *string         `json:"messageId"`
	Message         json.RawMessage `json:"message"`
	EnrichedMessage json.RawMessage `json:"enrichedMessage"`
}

// Store holds records in seq order. Records are immutable once appended, so
// readers get shared slices without copying.
type Store struct {
	serverID   string
	maxRecords int
	maxBytes   int
	done       chan struct{}
	closeOnce  sync.Once

	mu      sync.Mutex
	records []Record
	sizes   []int
	bytes   int
	evicted int
	through uint64
	cursor  uint64
	changed chan struct{}
}

// Option configures New.
type Option func(*Store)

// WithLimits replaces the default capacity.
func WithLimits(maxRecords, maxBytes int) Option {
	return func(s *Store) { s.maxRecords, s.maxBytes = maxRecords, maxBytes }
}

func New(serverID string, opts ...Option) *Store {
	s := &Store{serverID: serverID, maxRecords: DefaultMaxRecords, maxBytes: DefaultMaxBytes,
		changed: make(chan struct{}), done: make(chan struct{})}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Stats is the store occupancy. EvictedThrough is the highest evicted seq.
type Stats struct {
	Requests       int    `json:"requests"`
	Bytes          int    `json:"bytes"`
	Evicted        int    `json:"evicted"`
	EvictedThrough uint64 `json:"evictedThrough"`
}

func (s *Store) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Stats{Requests: len(s.records), Bytes: s.bytes, Evicted: s.evicted, EvictedThrough: s.through}
}

// Limits returns the capacity: records, then bytes.
func (s *Store) Limits() (int, int) { return s.maxRecords, s.maxBytes }

// Append stamps the record with the next seq, stores it and evicts the
// oldest records past the capacity.
func (s *Store) Append(r Record) Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cursor++
	r.RecordVersion = RecordVersion
	r.ServerID = s.serverID
	r.Seq = s.cursor
	size := recordBytes(r)
	s.records = append(s.records, r)
	s.sizes = append(s.sizes, size)
	s.bytes += size
	s.evict()
	close(s.changed)
	s.changed = make(chan struct{})
	return r
}

// evict drops the oldest records until both limits hold. The newest record
// always stays. Views share the backing array, so evicted slots are not
// cleared; the array is copied once it is mostly dead.
func (s *Store) evict() {
	n := 0
	for len(s.records)-n > 1 && (len(s.records)-n > s.maxRecords || s.bytes > s.maxBytes) {
		s.bytes -= s.sizes[n]
		s.through = s.records[n].Seq
		n++
	}
	if n == 0 {
		return
	}
	s.evicted += n
	s.records, s.sizes = s.records[n:], s.sizes[n:]
	if cap(s.records) > 2*len(s.records)+64 {
		s.records, s.sizes = slices.Clone(s.records), slices.Clone(s.sizes)
	}
}

// recordBytes estimates the memory a record holds: bodies, messages and
// header values.
func recordBytes(r Record) int {
	n := len(r.Request.Body) + len(r.Request.BodyBase64) + len(r.Request.Target) + len(r.Response.Body)
	for _, ev := range r.Events {
		n += len(ev.Message) + len(ev.EnrichedMessage)
	}
	for _, h := range []http.Header{r.Request.Headers, r.Response.Headers} {
		for k, vs := range h {
			n += len(k)
			for _, v := range vs {
				n += len(v)
			}
		}
	}
	return n
}

// Cursor is the highest seq assigned so far, 0 on an empty store.
func (s *Store) Cursor() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursor
}

// View is a consistent read of the store: the records after a seq, the cursor
// at read time, and a channel closed on the next Append.
type View struct {
	Records []Record
	Cursor  uint64
	Changed <-chan struct{}
}

func (s *Store) Since(seq uint64) View {
	s.mu.Lock()
	defer s.mu.Unlock()

	i := sort.Search(len(s.records), func(i int) bool { return s.records[i].Seq > seq })
	return View{
		Records: s.records[i:len(s.records):len(s.records)],
		Cursor:  s.cursor,
		Changed: s.changed,
	}
}

// Done is closed by Close so long-poll waiters can stop on shutdown.
func (s *Store) Done() <-chan struct{} { return s.done }

func (s *Store) Close() {
	s.closeOnce.Do(func() { close(s.done) })
}

// Get returns the record with this seq, if the store still holds it.
func (s *Store) Get(seq uint64) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	i := sort.Search(len(s.records), func(i int) bool { return s.records[i].Seq >= seq })
	if i == len(s.records) || s.records[i].Seq != seq {
		return Record{}, false
	}
	return s.records[i], true
}

// Reset removes every record and returns them. seq keeps counting, and
// waiters are not woken: a reset adds no match.
func (s *Store) Reset() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := s.records
	s.records, s.sizes, s.bytes = nil, nil, 0
	return removed
}
