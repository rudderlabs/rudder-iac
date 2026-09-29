// Package store keeps captured requests in memory and orders them by seq.
package store

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"
)

const RecordVersion = 1

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
	Method       string      `json:"method"`
	Target       string      `json:"target"`
	Headers      http.Header `json:"headers"`
	RemoteAddr   string      `json:"remoteAddr"`
	BodyEncoding string      `json:"bodyEncoding"`
	BodyBytes    int         `json:"bodyBytes"`
	Body         string      `json:"body"`
	BodyBase64   []byte      `json:"bodyBase64"`
	BodyComplete bool        `json:"bodyComplete"`
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
	serverID  string
	done      chan struct{}
	closeOnce sync.Once

	mu      sync.Mutex
	records []Record
	cursor  uint64
	changed chan struct{}
}

func New(serverID string) *Store {
	return &Store{serverID: serverID, changed: make(chan struct{}), done: make(chan struct{})}
}

// Append stamps the record with the next seq and stores it.
func (s *Store) Append(r Record) Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cursor++
	r.RecordVersion = RecordVersion
	r.ServerID = s.serverID
	r.Seq = s.cursor
	s.records = append(s.records, r)
	close(s.changed)
	s.changed = make(chan struct{})
	return r
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
