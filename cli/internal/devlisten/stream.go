package devlisten

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
)

// streamSink writes each accepted event as one JSON line on out, and one
// line per refused ingestion request on log. It keeps nothing, so a long run
// uses no more memory than a short one.
type streamSink struct {
	mu   sync.Mutex
	out  io.Writer
	log  io.Writer
	buf  bytes.Buffer
	line bytes.Buffer

	// serverID is set when each line carries a _dev object, so lines from
	// two listeners can be told apart. seq counts the lines written.
	serverID string
	seq      uint64

	// failed receives the first write error on out. The sink writes nothing
	// after it and counts the events it could not write in lost.
	failed  chan error
	failure error
	lost    int
}

func newStreamSink(out, log io.Writer, serverID string) *streamSink {
	return &streamSink{out: out, log: log, serverID: serverID, failed: make(chan error, 1)}
}

// Capture runs before the answer is written, so a slow reader of out slows
// the senders down instead of growing a queue.
// ponytail: one mutex for all writers; a pipe reader is the bottleneck anyway.
func (s *streamSink) Capture(c *ingest.Capture) {
	if c.Kind != "ingestion" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// The same rule as the query API: a pixel request answers 200 but drops
	// its event when Rejection is set.
	if c.StatusCode < 200 || c.StatusCode > 299 || c.Rejection != nil {
		reason := http.StatusText(c.StatusCode)
		if c.Rejection != nil && c.Rejection.Reason != "" {
			reason = c.Rejection.Reason
		}
		fmt.Fprintf(s.log, "refused: %d %s %s: %s\n", c.StatusCode, c.Request.Method, c.Route, reason)
		return
	}
	if s.failure != nil {
		s.lost += len(c.Events)
		return
	}
	s.buf.Reset()
	for _, ev := range c.Events {
		s.line.Reset()
		// Compact only drops whitespace, so each line is the event the SDK
		// sent, and a pretty-printed body still gives one event per line.
		if err := json.Compact(&s.line, ev.Message); err != nil {
			s.line.Write(bytes.ReplaceAll(ev.Message, []byte("\n"), nil))
		}
		s.stamp(c.ReceivedAt)
		s.buf.WriteByte('\n')
	}
	if n, err := s.out.Write(s.buf.Bytes()); err != nil {
		// The SDK still gets its answer, so the loss must stop the listener
		// and show on stderr instead. A write that failed part way, as on a
		// disk that filled, still wrote the whole lines before n, and each
		// line holds one event.
		written := bytes.Count(s.buf.Bytes()[:n], []byte{'\n'})
		s.failure, s.lost = err, len(c.Events)-written
		s.failed <- err
	}
}

// stamp writes the line to buf. With a server id, a _dev object goes first
// inside the event, and the rest of the line is the event as it is.
func (s *streamSink) stamp(receivedAt time.Time) {
	event := s.line.Bytes()
	if s.serverID == "" || len(event) < 2 || event[0] != '{' {
		s.buf.Write(event)
		return
	}
	s.seq++
	s.buf.WriteString(`{"_dev":{"serverId":`)
	s.buf.WriteString(strconv.Quote(s.serverID))
	s.buf.WriteString(`,"seq":`)
	s.buf.WriteString(strconv.FormatUint(s.seq, 10))
	s.buf.WriteString(`,"receivedAt":"`)
	s.buf.WriteString(receivedAt.UTC().Format(time.RFC3339Nano))
	s.buf.WriteString(`"}`)
	if rest := event[1:]; rest[0] != '}' {
		s.buf.WriteByte(',')
		s.buf.Write(rest)
		return
	}
	s.buf.WriteByte('}')
}

// Lost is the number of accepted events the sink could not write.
func (s *streamSink) Lost() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lost
}

// fanOut hands each capture to every sink in order.
type fanOut []ingest.Sink

func (f fanOut) Capture(c *ingest.Capture) {
	for _, s := range f {
		s.Capture(c)
	}
}
