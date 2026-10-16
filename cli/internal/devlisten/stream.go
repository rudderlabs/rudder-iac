package devlisten

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
)

// streamSink writes each accepted event as one JSON line on out, and one
// line per refused ingestion request on log. It keeps nothing, so a long run
// uses no more memory than a short one.
type streamSink struct {
	mu  sync.Mutex
	out io.Writer
	log io.Writer
	buf bytes.Buffer
}

func newStreamSink(out, log io.Writer) *streamSink {
	return &streamSink{out: out, log: log}
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
	s.buf.Reset()
	for _, ev := range c.Events {
		// Compact only drops whitespace, so each line is the event the SDK
		// sent, and a pretty-printed body still gives one event per line.
		if err := json.Compact(&s.buf, ev.Message); err != nil {
			s.buf.Write(bytes.ReplaceAll(ev.Message, []byte("\n"), nil))
		}
		s.buf.WriteByte('\n')
	}
	_, _ = s.out.Write(s.buf.Bytes())
}

// fanOut hands each capture to every sink in order.
type fanOut []ingest.Sink

func (f fanOut) Capture(c *ingest.Capture) {
	for _, s := range f {
		s.Capture(c)
	}
}
