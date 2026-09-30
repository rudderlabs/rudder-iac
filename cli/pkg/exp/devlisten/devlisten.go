// Package devlisten runs a local capture server for the events an app sends
// to RudderStack, and a client for its query API. It uses the standard
// library only and never imports cli/internal, so a Go test can import it
// without a token or the experimental flag (contract section 7). The API is
// unstable while it lives under pkg/exp.
package devlisten

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/api"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/ingest"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

const (
	DefaultBind     = "127.0.0.1"
	DefaultWriteKey = "dev"

	closeOnCancelTimeout = 5 * time.Second
)

// Stop reasons reported by Server.StopReason.
const (
	StopReasonClose = "close"
	StopReasonStop  = "stop"
	StopReasonIdle  = "idle"
)

// ErrPortInUse is returned by Start when a fixed port is taken.
var ErrPortInUse = errors.New("port in use")

type config struct {
	port      int
	bind      string
	idleExit  time.Duration
	onCapture func(Capture)
}

// Option configures Start. Start validates the combined settings.
type Option func(*config)

// WithPort fixes the port. 0, the default, lets the OS pick a free one.
func WithPort(port int) Option { return func(c *config) { c.port = port } }

// WithBind sets the listen address. The default is 127.0.0.1.
func WithBind(bind string) Option { return func(c *config) { c.bind = bind } }

// WithIdleExit stops the server after d without activity. Captures, query
// calls and open long-polls count as activity; the index and /info do not,
// so a discovery probe cannot keep a forgotten server alive. 0 turns it off.
func WithIdleExit(d time.Duration) Option { return func(c *config) { c.idleExit = d } }

// WithCaptureHook calls f after every captured request is answered.
func WithCaptureHook(f func(Capture)) Option { return func(c *config) { c.onCapture = f } }

// Capture describes one captured request for a progress line. Type and
// Event are the common value of the request's events, "mixed" when they
// differ, and "" when absent.
type Capture struct {
	Seq        uint64
	Kind       string
	Route      string
	Method     string
	StatusCode int
	Outcome    string
	Stage      string
	Type       string
	Event      string
	Events     int
}

// Ready is the object `dev listen` prints once on stdout after the bind
// (contract section 6.1).
type Ready struct {
	Ready          bool      `json:"ready"`
	APIVersion     string    `json:"apiVersion"`
	ServerID       string    `json:"serverId"`
	URL            string    `json:"url"`
	Port           int       `json:"port"`
	Bind           string    `json:"bind"`
	PID            int       `json:"pid"`
	StartedAt      time.Time `json:"startedAt"`
	WriteKey       string    `json:"writeKey"`
	WriteKeyPolicy string    `json:"writeKeyPolicy"`
	Cursor         uint64    `json:"cursor"`
	StateFile      *string   `json:"stateFile"`
}

// Parameter is one query parameter of a /_dev/v1 route.
type Parameter struct {
	Name       string
	Default    string
	Repeatable bool
}

// Parameters lists the query parameters of each route: "events",
// "requests", "requests/{seq}", "summary" and "info".
func Parameters() map[string][]Parameter {
	out := make(map[string][]Parameter, len(api.Params))
	for route, params := range api.Params {
		for _, p := range params {
			out[route] = append(out[route], Parameter(p))
		}
	}
	return out
}

type Server struct {
	id       api.Identity
	store    *store.Store
	gateway  *ingest.Gateway
	http     *http.Server
	served   chan error
	closeErr error
	close    sync.Once
	done     chan struct{}
	reason   atomic.Value

	idleExit   time.Duration
	lastActive atomic.Int64
	inflight   atomic.Int64

	connMu sync.Mutex
	fresh  map[net.Conn]struct{}
}

// Start binds, starts serving and returns. Cancelling ctx closes the server
// with a 5 s deadline.
func Start(ctx context.Context, opts ...Option) (*Server, error) {
	cfg := config{bind: DefaultBind}
	for _, opt := range opts {
		opt(&cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	ln, err := listen(ctx, cfg)
	if err != nil {
		return nil, err
	}

	port := ln.Addr().(*net.TCPAddr).Port
	id := api.Identity{
		APIVersion:     api.APIVersion,
		ServerID:       newServerID(),
		URL:            "http://" + net.JoinHostPort(urlHost(cfg.bind), strconv.Itoa(port)),
		Port:           port,
		Bind:           cfg.bind,
		PID:            os.Getpid(),
		StartedAt:      time.Now().UTC().Truncate(time.Second),
		WriteKey:       DefaultWriteKey,
		WriteKeyPolicy: "any",
	}

	st := store.New(id.ServerID)
	s := &Server{
		id:       id,
		store:    st,
		gateway:  ingest.New(st, id.StartedAt),
		served:   make(chan error, 1),
		done:     make(chan struct{}),
		idleExit: cfg.idleExit,
		fresh:    map[net.Conn]struct{}{},
	}
	s.touch()
	s.setCaptureHook(cfg.onCapture)
	queryAPI := api.New(st, id, api.Config{
		CheckHost: api.IsLoopback(cfg.bind),
		Shutdown:  func() { s.stop(StopReasonStop) },
	})
	s.http = &http.Server{
		Handler:           s.handler(queryAPI),
		ReadHeaderTimeout: 10 * time.Second,
		ConnState:         s.trackConn,
	}

	go func() { s.served <- s.http.Serve(ln) }()
	if s.idleExit > 0 {
		go s.watchIdle()
	}
	context.AfterFunc(ctx, func() { s.stop(StopReasonClose) })
	return s, nil
}

func (c config) validate() error {
	if c.port < 0 || c.port > 65535 || c.idleExit < 0 {
		return fmt.Errorf("invalid options: port %d, idle exit %s", c.port, c.idleExit)
	}
	return nil
}

func (s *Server) setCaptureHook(f func(Capture)) {
	if f != nil {
		s.gateway.OnCapture(func(rec store.Record) { f(newCapture(rec)) })
	}
}

func listen(ctx context.Context, cfg config) (net.Listener, error) {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort(cfg.bind, strconv.Itoa(cfg.port)))
	if errors.Is(err, syscall.EADDRINUSE) {
		return nil, fmt.Errorf("listening on %s:%d: %w", cfg.bind, cfg.port, ErrPortInUse)
	}
	if err != nil {
		return nil, fmt.Errorf("listening on %s:%d: %w", cfg.bind, cfg.port, err)
	}
	return ln, nil
}

// handler routes /_dev/ to the query API and the rest to ingestion, and
// counts every non-exempt request as activity for the idle timer.
func (s *Server) handler(queryAPI http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if resetsIdle(r.URL.Path) {
			s.inflight.Add(1)
			defer func() { s.inflight.Add(-1); s.touch() }()
		}
		if strings.HasPrefix(r.URL.Path, api.Prefix) {
			queryAPI.ServeHTTP(w, r)
			return
		}
		s.gateway.ServeHTTP(w, r)
	})
}

// idleExempt are the paths a healthcheck or discovery probe calls.
var idleExempt = map[string]bool{
	"/": true, "/health": true, "/internal/readiness": true, "/version": true,
	"/_dev/v1/": true, "/_dev/v1": true, "/_dev/v1/info": true, "/_dev/v1/openapi.yaml": true,
}

func resetsIdle(path string) bool { return !idleExempt[path] }

func (s *Server) touch() { s.lastActive.Store(time.Now().UnixNano()) }

func (s *Server) watchIdle() {
	ticker := time.NewTicker(min(s.idleExit/4, time.Second))
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			idle := time.Since(time.Unix(0, s.lastActive.Load()))
			if s.inflight.Load() == 0 && idle >= s.idleExit {
				s.stop(StopReasonIdle)
				return
			}
		}
	}
}

// stop closes the server with the default deadline and records why.
func (s *Server) stop(reason string) {
	s.reason.CompareAndSwap(nil, reason)
	ctx, cancel := context.WithTimeout(context.Background(), closeOnCancelTimeout)
	defer cancel()
	_ = s.Close(ctx)
}

func newCapture(rec store.Record) Capture {
	c := Capture{Seq: rec.Seq, Kind: rec.Kind, Route: rec.Route, Method: rec.Request.Method,
		StatusCode: rec.StatusCode, Outcome: rec.Outcome, Events: len(rec.Events)}
	if rec.Rejection != nil {
		c.Stage = rec.Rejection.Stage
	}
	for i, ev := range rec.Events {
		c.Type = common(i, c.Type, ev.Type)
		c.Event = common(i, c.Event, ev.Event)
	}
	return c
}

func common(i int, acc string, v *string) string {
	val := ""
	if v != nil {
		val = *v
	}
	if i > 0 && val != acc {
		return "mixed"
	}
	return val
}

// IsLoopback reports whether a bind address only accepts local connections.
func IsLoopback(bind string) bool { return api.IsLoopback(bind) }

// urlHost keeps the URL reachable from this host: a wildcard bind is
// reported as 127.0.0.1.
func urlHost(bind string) string {
	switch bind {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	}
	return bind
}

// newServerID is 64 random bits as 16 hex characters.
func newServerID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (s *Server) URL() string { return s.id.URL }

func (s *Server) Port() int { return s.id.Port }

// Ready returns the ready object with the store cursor at call time.
func (s *Server) Ready() Ready {
	return Ready{
		Ready:          true,
		APIVersion:     s.id.APIVersion,
		ServerID:       s.id.ServerID,
		URL:            s.id.URL,
		Port:           s.id.Port,
		Bind:           s.id.Bind,
		PID:            s.id.PID,
		StartedAt:      s.id.StartedAt,
		WriteKey:       s.id.WriteKey,
		WriteKeyPolicy: s.id.WriteKeyPolicy,
		Cursor:         s.store.Cursor(),
	}
}

// Client returns a client for this server, pinned to its serverId.
func (s *Server) Client() *Client {
	return NewClient(s.id.URL, WithServerID(s.id.ServerID))
}

// Close stops the server: health routes answer 503, long-polls wake with
// 503 shutting_down, then the HTTP server drains until ctx ends.
func (s *Server) Close(ctx context.Context) error {
	s.close.Do(func() {
		s.reason.CompareAndSwap(nil, StopReasonClose)
		s.gateway.Stop()
		s.store.Close()
		s.http.SetKeepAlivesEnabled(false)
		s.closeFreshConns()
		if err := s.http.Shutdown(ctx); err != nil {
			s.closeErr = fmt.Errorf("shutting down: %w", err)
		}
		close(s.done)
	})
	return s.closeErr
}

// trackConn keeps the connections that have not sent a request yet.
// Shutdown treats such a connection as busy for 5 s, so a client's
// speculative dial would stall every stop.
func (s *Server) trackConn(c net.Conn, state http.ConnState) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if state == http.StateNew {
		s.fresh[c] = struct{}{}
		return
	}
	delete(s.fresh, c)
}

func (s *Server) closeFreshConns() {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	for c := range s.fresh {
		_ = c.Close()
	}
}

// Done is closed once the server has stopped, whatever stopped it.
func (s *Server) Done() <-chan struct{} { return s.done }

// StopReason is StopReasonClose, StopReasonStop or StopReasonIdle once Done
// is closed, and "" before.
func (s *Server) StopReason() string {
	reason, _ := s.reason.Load().(string)
	return reason
}

// Wait blocks until the server stops serving. It returns nil after Close.
func (s *Server) Wait() error {
	err := <-s.served
	s.served <- err
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
