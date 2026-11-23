package devlisten

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/api"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ui"
)

const (
	DefaultBind = "127.0.0.1"

	// maxHeaderBytes and idleTimeout are the rudder-server gateway defaults.
	// WriteTimeout stays 0, because a long-poll answers only when events arrive.
	maxHeaderBytes    = 512 << 10
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 720 * time.Second
	// readTimeout ends the read of a body that no handler consumes, such as
	// the body of a 405. net/http clears it once the body is read, so a GET
	// long-poll runs past it. The ingestion body deadline overrides it.
	readTimeout = 10 * time.Second
)

// ErrPortInUse is returned by Start when a fixed port is taken.
var ErrPortInUse = errors.New("port in use")

type Config struct {
	// Port 0 lets the system pick a free port.
	Port int
	// Bind defaults to DefaultBind.
	Bind string
	// WriteKeys is the allowlist; empty accepts every key.
	WriteKeys []string
	// AllowHosts are extra Host names the query API accepts.
	AllowHosts []string
	// Version is the CLI version that /info and /version report.
	Version string
	// Store keeps every request in memory and serves the query API and the
	// review page under /_dev/. Off, the listener keeps nothing.
	Store bool
	// Out gets each accepted event as one JSON line; nil discards them.
	Out io.Writer
	// Stamp adds a _dev object with serverId, seq and receivedAt to each
	// line on Out, for a file that must say which listener wrote it. It goes
	// first, or last when the event has a _dev of its own.
	Stamp bool
	// Log gets one line per refused ingestion request; nil discards them.
	Log io.Writer

	// readTimeout lets a test shorten the request read limit.
	readTimeout time.Duration
	// beforeServe lets a test hold a handler past the drain deadline.
	beforeServe func(*http.Request)
}

// Ready is the line sources listen prints once it accepts connections. A script
// reads the URL, the cursor and the pid from it.
type Ready struct {
	Ready bool `json:"ready"`
	api.Identity
	// Cursor and UI are set with --store only, so a script tells the modes
	// apart by ui.
	Cursor *uint64 `json:"cursor,omitempty"`
	UI     string  `json:"ui,omitempty"`
}

type Server struct {
	id     api.Identity
	store  *store.Store
	ingest *ingest.Handler
	api    *api.Handler
	http   *http.Server
	served chan error
	stream *streamSink

	beforeServe func(*http.Request)

	closeOnce sync.Once
	closeErr  error

	connMu sync.Mutex
	// conns holds every connection that is not idle or closed, and whether
	// it has sent a byte yet.
	conns   map[net.Conn]http.ConnState
	closing bool
}

// Start binds and serves in the background until Close.
func Start(cfg Config) (*Server, error) {
	if cfg.Bind == "" {
		cfg.Bind = DefaultBind
	}
	if cfg.readTimeout == 0 {
		cfg.readTimeout = readTimeout
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Port)))
	if errors.Is(err, syscall.EADDRINUSE) {
		return nil, fmt.Errorf("listening on %s port %d: %w", cfg.Bind, cfg.Port, ErrPortInUse)
	}
	if err != nil {
		return nil, fmt.Errorf("listening on %s port %d: %w", cfg.Bind, cfg.Port, err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	id := api.Identity{
		APIVersion:     "v1",
		ServerID:       newServerID(),
		URL:            "http://" + net.JoinHostPort(urlHost(cfg.Bind), strconv.Itoa(port)),
		Port:           port,
		Bind:           cfg.Bind,
		PID:            os.Getpid(),
		StartedAt:      time.Now().UTC().Truncate(time.Second),
		WriteKey:       "dev",
		WriteKeyPolicy: "any",
	}
	masked := make([]string, len(cfg.WriteKeys))
	for i, key := range cfg.WriteKeys {
		masked[i] = store.MaskWriteKey(key).Key
	}
	if len(masked) > 0 {
		id.WriteKey, id.WriteKeyPolicy = masked[0], "allowlist"
	}

	stampID := ""
	if cfg.Stamp {
		stampID = id.ServerID
	}
	stream := newStreamSink(orDiscard(cfg.Out), orDiscard(cfg.Log), stampID)
	var sink ingest.Sink = stream
	var st *store.Store
	var apiHandler *api.Handler
	if cfg.Store {
		st = store.New()
		// The store goes first, so a reader that sees a line can query it.
		sink = fanOut{st, sink}
		apiHandler = api.New(st, api.Config{
			Identity: id, WriteKeys: masked, AllowHosts: cfg.AllowHosts, Version: cfg.Version, Guide: Guide,
			UI: ui.Handler(api.UIPath),
		})
	}
	s := &Server{
		id:     id,
		store:  st,
		ingest: ingest.New(sink, cfg.WriteKeys, cfg.Version),
		api:    apiHandler,
		served: make(chan error, 1),
		stream: stream,
		conns:  map[net.Conn]http.ConnState{},

		beforeServe: cfg.beforeServe,
	}
	s.http = &http.Server{
		Handler:           s,
		MaxHeaderBytes:    maxHeaderBytes,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       cfg.readTimeout,
		IdleTimeout:       idleTimeout,
		ConnState:         s.trackConn,
	}
	go func() {
		err := s.http.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		s.served <- err
	}()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.beforeServe != nil {
		s.beforeServe(r)
	}
	if strings.HasPrefix(r.URL.Path, api.Prefix) {
		if s.api == nil {
			storeDisabled(w)
			return
		}
		s.api.ServeHTTP(w, r)
		return
	}
	s.ingest.ServeHTTP(w, r)
}

// Ready returns the ready object with the cursor at call time.
func (s *Server) Ready() Ready {
	r := Ready{Ready: true, Identity: s.id}
	if s.store != nil {
		cursor := s.store.Cursor()
		r.Cursor, r.UI = &cursor, s.id.URL+api.UIPath
	}
	return r
}

// storeDisabled answers the query API of a listener started without
// --store, in the error shape of the query API.
func storeDisabled(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.WriteString(w, `{"error":{"status":404,"code":"store_disabled",`+
		`"message":"this listener keeps nothing: it prints events on stdout. Restart it with --store to read them here",`+
		`"param":null,"details":null,"next":"rudder-cli sources listen --store"}}`+"\n")
}

func orDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

// Done receives the error that ended serving, nil after Close.
func (s *Server) Done() <-chan error {
	return s.served
}

// OutFailed receives the first error writing an event to Out. The listener
// keeps answering after it, so the caller must stop it.
func (s *Server) OutFailed() <-chan error {
	return s.stream.failed
}

// Lost is the number of accepted events that never reached Out.
func (s *Server) Lost() int {
	return s.stream.Lost()
}

// Close drains the server until ctx ends, then drops every connection that
// is left. Captures live in memory only, so an upload cut at shutdown loses
// nothing that would outlive the process.
func (s *Server) Close(ctx context.Context) error {
	s.closeOnce.Do(func() {
		s.ingest.Stop()
		if s.api != nil {
			s.api.Stop()
		}
		s.expireReads()
		if err := s.http.Shutdown(ctx); err != nil {
			_ = s.http.Close()
			s.closeErr = fmt.Errorf("draining connections: %w", err)
		}
	})
	return s.closeErr
}

// expireReads ends every read in progress. Shutdown waits for handlers, and
// a stalled upload would hold one until its 10 s body deadline. It counts a
// connection that has sent nothing as busy for 5 s, so those are closed.
func (s *Server) expireReads() {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	s.closing = true
	for c, state := range s.conns {
		if state == http.StateNew {
			_ = c.Close()
			continue
		}
		_ = c.SetReadDeadline(time.Now())
	}
}

func (s *Server) trackConn(c net.Conn, state http.ConnState) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	switch {
	case state == http.StateNew && s.closing:
		_ = c.Close()
	case state == http.StateNew || state == http.StateActive:
		s.conns[c] = state
	default:
		delete(s.conns, c)
	}
}

func (s *Server) activeConns() int {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	return len(s.conns)
}

// readingConns counts the connections that have sent a byte, so a test can
// tell a held request from a connection that expireReads closes unread.
func (s *Server) readingConns() int {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	n := 0
	for _, state := range s.conns {
		if state == http.StateActive {
			n++
		}
	}
	return n
}

// urlHost keeps the URL reachable from this host: a wildcard bind answers
// on loopback too.
func urlHost(bind string) string {
	if ip := net.ParseIP(bind); ip != nil && ip.IsUnspecified() {
		return "127.0.0.1"
	}
	return bind
}

// newServerID tells two listeners, or one listener before and after a
// restart, apart.
func newServerID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
