// Package ingest serves the ingestion and SDK bootstrap routes. It copies
// rudder-ingestion-svc (the oracle) and captures every request it answers.
package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// ingestRoutes maps each POST ingestion route to its request type
// (SVC internal/gateway/gateway.go:573-611). The reqType enricher writes the
// type into single-event payloads.
var ingestRoutes = map[string]string{
	"/v1/alias":        "alias",
	"/v1/batch":        "batch",
	"/v1/group":        "group",
	"/v1/identify":     "identify",
	"/v1/page":         "page",
	"/v1/screen":       "screen",
	"/v1/track":        "track",
	"/beacon/v1/batch": "batch",
}

const probeUserAgentPrefix = "rudder-cli dev send/"

type Gateway struct {
	store     *store.Store
	startedAt time.Time
	now       func() time.Time
	newUUID   func() string
	stopping  atomic.Bool
	onCapture func(store.Record)
}

func New(st *store.Store, startedAt time.Time) *Gateway {
	return &Gateway{store: st, startedAt: startedAt, now: time.Now, newUUID: newUUIDv4}
}

// OnCapture sets a function called with every stored record, after the
// response is written. Call it before serving.
func (g *Gateway) OnCapture(f func(store.Record)) { g.onCapture = f }

// Stop makes the health routes answer 503 while the server drains.
func (g *Gateway) Stop() { g.stopping.Store(true) }

// reply is the decided response plus what the capture record needs.
type reply struct {
	status    int
	header    http.Header
	body      []byte
	kind      string
	transport string
	writeKey  string
	decoded   []byte
	events    []store.Event
	rejection *store.Rejection
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	receivedAt := g.now().UTC()
	// SVC rewriteDoubleSlashMiddleware (gateway.go:452-459).
	if strings.HasPrefix(r.URL.Path, "//") {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/")
	}

	if rep, ok := g.health(r, r.URL.Path); ok {
		withActualCORS(r, rep.header)
		writeReply(w, rep)
		return
	}

	raw, complete := readBody(r)
	rep := g.route(r, raw, receivedAt)

	// The record is stored before the response is written, so a sender that
	// has read the response always finds its request in the store.
	rec := g.store.Append(g.record(r, rep, raw, complete, receivedAt))
	writeReply(w, rep)
	if g.onCapture != nil {
		g.onCapture(rec)
	}
}

func (g *Gateway) route(r *http.Request, raw []byte, receivedAt time.Time) reply {
	if isPreflight(r) {
		return preflight(r)
	}

	var rep reply
	path := r.URL.Path
	reqType, isIngest := ingestRoutes[path]
	switch {
	case isIngest && r.Method == http.MethodPost:
		rep = g.ingest(r, reqType, transportOf(path), raw, receivedAt)
	case isIngest:
		rep = methodNotAllowed("ingestion", transportOf(path), http.MethodPost)
	case isSourceConfig(r):
		rep = g.sourceConfig(r)
	default:
		rep = unknownPath()
	}
	withActualCORS(r, rep.header)
	return rep
}

func isSourceConfig(r *http.Request) bool {
	path := r.URL.Path
	return (path == "/sourceConfig" || path == "/sourceConfig/") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead)
}

func transportOf(path string) string {
	if strings.HasPrefix(path, "/beacon/") {
		return "beacon"
	}
	return "http"
}

func readBody(r *http.Request) ([]byte, bool) {
	if r.Body == nil {
		return nil, true
	}
	raw, err := io.ReadAll(r.Body)
	return raw, err == nil
}

func writeReply(w http.ResponseWriter, rep reply) {
	for k, v := range rep.header {
		w.Header()[k] = v
	}
	w.WriteHeader(rep.status)
	_, _ = w.Write(rep.body)
}

func (g *Gateway) record(r *http.Request, rep reply, raw []byte, complete bool, receivedAt time.Time) store.Record {
	headers, redacted := redact(r.Header)
	rec := store.Record{
		Kind:       rep.kind,
		Probe:      strings.HasPrefix(r.UserAgent(), probeUserAgentPrefix),
		ReceivedAt: receivedAt,
		Route:      routeOf(r.URL.Path),
		Transport:  rep.transport,
		StatusCode: rep.status,
		Outcome:    "accepted",
		WriteKey:   rep.writeKey,
		SourceID:   SourceID(rep.writeKey),
		Rejection:  rep.rejection,
		Request: store.Request{
			Method:          r.Method,
			Target:          r.RequestURI,
			Headers:         headers,
			RedactedHeaders: redacted,
			RemoteAddr:      r.RemoteAddr,
			BodyEncoding:    r.Header.Get("Content-Encoding"),
			BodyBytes:       len(raw),
			Body:            string(raw),
			BodyComplete:    complete,
		},
		Response: store.Response{StatusCode: rep.status, Headers: rep.header.Clone(), Body: string(rep.body)},
		Events:   rep.events,
	}
	if rec.Request.BodyEncoding != "" {
		rec.Request.Body = string(rep.decoded)
		rec.Request.BodyBase64 = raw
	}
	if rec.Events == nil {
		rec.Events = []store.Event{}
	}
	if rep.rejection != nil || rep.status < 200 || rep.status > 299 {
		rec.Outcome = "rejected"
		rec.Failed = true
	}
	return rec
}

// credentialHeader matches header names that carry credentials. Cookies are
// not scoped by port, so a browser sends the session cookies of other
// localhost apps too.
var credentialHeader = regexp.MustCompile(`(?i)(token|secret|api-?key|auth)`)

// routeOf is the path without a trailing slash; request.target keeps it.
// Summary counts and route filters use this form.
func routeOf(path string) string {
	if len(path) > 1 {
		return strings.TrimSuffix(path, "/")
	}
	return path
}

// redact replaces credential header values and returns the redacted names,
// sorted.
func redact(h http.Header) (http.Header, []string) {
	out := h.Clone()
	names := []string{}
	for name := range out {
		if name == "Cookie" || credentialHeader.MatchString(name) {
			out[name] = []string{"REDACTED"}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return out, names
}

// SourceID is the source id the server reports for a write key: "dev-" plus
// the first 12 hex characters of sha256(writeKey).
func SourceID(writeKey string) string {
	if writeKey == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(writeKey))
	return "dev-" + hex.EncodeToString(sum[:])[:12]
}
