// Package devlisten runs a local capture server for the events an app sends
// to RudderStack, and a client for its query API. It uses the standard
// library only and never imports cli/internal, so a Go test can import it
// without a token or the experimental flag (contract section 7).
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

// ErrPortInUse is returned by Start when a fixed port is taken.
var ErrPortInUse = errors.New("port in use")

type config struct {
	port int
	bind string
}

type Option func(*config)

// WithPort fixes the port. 0, the default, lets the OS pick a free one.
func WithPort(port int) Option { return func(c *config) { c.port = port } }

// WithBind sets the listen address. The default is 127.0.0.1.
func WithBind(bind string) Option { return func(c *config) { c.bind = bind } }

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

type Server struct {
	id       api.Identity
	store    *store.Store
	gateway  *ingest.Gateway
	http     *http.Server
	served   chan error
	closeErr error
	close    sync.Once
}

// Start binds, starts serving and returns. Cancelling ctx closes the server
// with a 5 s deadline.
func Start(ctx context.Context, opts ...Option) (*Server, error) {
	cfg := config{bind: DefaultBind}
	for _, opt := range opts {
		opt(&cfg)
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort(cfg.bind, strconv.Itoa(cfg.port)))
	if errors.Is(err, syscall.EADDRINUSE) {
		return nil, fmt.Errorf("listening on %s:%d: %w", cfg.bind, cfg.port, ErrPortInUse)
	}
	if err != nil {
		return nil, fmt.Errorf("listening on %s:%d: %w", cfg.bind, cfg.port, err)
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
		id:      id,
		store:   st,
		gateway: ingest.New(st, id.StartedAt),
		served:  make(chan error, 1),
	}
	queryAPI := api.New(st, id)
	s.http = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, api.Prefix) {
				queryAPI.ServeHTTP(w, r)
				return
			}
			s.gateway.ServeHTTP(w, r)
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() { s.served <- s.http.Serve(ln) }()
	context.AfterFunc(ctx, func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), closeOnCancelTimeout)
		defer cancel()
		_ = s.Close(closeCtx)
	})
	return s, nil
}

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
		s.gateway.Stop()
		s.store.Close()
		if err := s.http.Shutdown(ctx); err != nil {
			s.closeErr = fmt.Errorf("shutting down: %w", err)
		}
	})
	return s.closeErr
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
