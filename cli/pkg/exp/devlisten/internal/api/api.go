// Package api serves the query API under /_dev/v1/ (contract section 4).
package api

import (
	"encoding/json"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

const (
	APIVersion = "v1"
	Prefix     = "/_dev/"
	base       = "/_dev/v1/"
)

// Identity is the identity core shared by the ready line and /info
// (contract section 4.1).
type Identity struct {
	APIVersion     string    `json:"apiVersion"`
	ServerID       string    `json:"serverId"`
	URL            string    `json:"url"`
	Port           int       `json:"port"`
	Bind           string    `json:"bind"`
	PID            int       `json:"pid"`
	StartedAt      time.Time `json:"startedAt"`
	WriteKey       string    `json:"writeKey"`
	WriteKeyPolicy string    `json:"writeKeyPolicy"`
}

// Config holds the handler settings that depend on how the server runs.
type Config struct {
	// CheckHost refuses a /_dev/ request whose Host is not a loopback name
	// with the server port. It blocks DNS rebinding on a loopback bind.
	CheckHost bool
	// Shutdown is called once after POST /shutdown is answered.
	Shutdown func()
}

type Handler struct {
	store  *store.Store
	id     Identity
	cfg    Config
	now    func() time.Time
	routes map[string]route
	// waiting, when set, is called each time a long-poll starts to wait.
	// Tests use it to append only after the poll is registered.
	waiting func()
}

func New(st *store.Store, id Identity, cfg Config) *Handler {
	h := &Handler{store: st, id: id, cfg: cfg, now: time.Now}
	h.routes = map[string]route{
		base:                          {http.MethodGet, h.index},
		strings.TrimSuffix(base, "/"): {http.MethodGet, h.index},
		base + "info":                 {http.MethodGet, h.info},
		base + "events":               {http.MethodGet, h.events},
		base + "requests":             {http.MethodGet, h.requests},
		base + "summary":              {http.MethodGet, h.summary},
		base + "reset":                {http.MethodPost, h.reset},
		base + "shutdown":             {http.MethodPost, h.shutdown},
	}
	return h
}

type route struct {
	method string
	handle http.HandlerFunc
}

func (h *Handler) route(path string) (route, bool) {
	if rt, ok := h.routes[path]; ok {
		return rt, true
	}
	if strings.HasPrefix(path, base+"requests/") {
		return route{http.MethodGet, h.request}, true
	}
	return route{}, false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if err := h.guard(r); err != nil {
		writeError(w, *err)
		return
	}
	rt, ok := h.route(r.URL.Path)
	if !ok {
		writeError(w, apiError{status: http.StatusNotFound, Code: "not_found", Message: "no route " + r.URL.Path,
			Next: strp(h.curlIndex())})
		return
	}
	if r.Method != rt.method {
		w.Header().Set("Allow", rt.method)
		writeError(w, apiError{status: http.StatusMethodNotAllowed, Code: "method_not_allowed",
			Message: r.Method + " is not allowed on " + r.URL.Path, Next: strp(h.curlIndex())})
		return
	}
	if rt.method == http.MethodPost && !isJSON(r.Header.Get("Content-Type")) {
		writeError(w, apiError{status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type",
			Message: "POST needs Content-Type: application/json", Next: strp(h.curlIndex())})
		return
	}
	rt.handle(w, r)
}

// guard runs the checks every /_dev/ request passes first: Host, browser
// origin, then shutdown.
func (h *Handler) guard(r *http.Request) *apiError {
	if h.cfg.CheckHost && !h.allowedHost(r.Host) {
		return &apiError{status: http.StatusForbidden, Code: "host_not_allowed",
			Message: "Host " + r.Host + " is not a loopback name for this server", Next: strp(nextShell)}
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "none" && site != "same-origin" {
		return &apiError{status: http.StatusForbidden, Code: "browser_origin",
			Message: "the query API refuses cross-site browser requests", Next: strp(nextShell)}
	}
	select {
	case <-h.store.Done():
		return &errShuttingDown
	default:
		return nil
	}
}

func (h *Handler) allowedHost(host string) bool {
	port := strconv.Itoa(h.id.Port)
	for _, name := range []string{"127.0.0.1", "::1", "localhost"} {
		if host == net.JoinHostPort(name, port) {
			return true
		}
	}
	return false
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

func (h *Handler) curlIndex() string {
	return "curl -fsS " + shellQuote(h.id.URL+base)
}

type index struct {
	APIVersion  string            `json:"apiVersion"`
	ServerID    string            `json:"serverId"`
	Description string            `json:"description"`
	Next        string            `json:"next"`
	Curl        string            `json:"curl"`
	Links       map[string]string `json:"links"`
	Help        string            `json:"help"`
}

func (h *Handler) index(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, index{
		APIVersion:  APIVersion,
		ServerID:    h.id.ServerID,
		Description: "Inspect local captures. Start with summary.",
		Next:        "rudder-cli dev summary --since 0 --json",
		Curl:        "curl -fsS " + shellQuote(h.id.URL+base+"summary?serverId="+h.id.ServerID),
		Links: map[string]string{"summary": "summary", "events": "events?view=summary", "requests": "requests",
			"request": "requests/{seq}", "info": "info"},
		Help: "rudder-cli dev --help",
	})
}

type info struct {
	Ready bool `json:"ready"`
	Identity
	WriteKeys     []string  `json:"writeKeys"`
	RecordVersion int       `json:"recordVersion"`
	Cursor        uint64    `json:"cursor"`
	Store         infoStore `json:"store"`
}

type storeStats struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
	Control  int `json:"control"`
}

// infoStore adds the capacity and eviction counts to the /info store object.
type infoStore struct {
	storeStats
	Bytes          int    `json:"bytes"`
	Evicted        int    `json:"evicted"`
	EvictedThrough uint64 `json:"evictedThrough"`
	MaxRequests    int    `json:"maxRequests"`
	MaxBytes       int    `json:"maxBytes"`
}

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	if _, err := parseParams(r.URL.Query(), "info"); err != nil {
		writeError(w, *err)
		return
	}
	view := h.store.Since(0)
	counts := countRecords(view.Records)
	stats := h.store.Stats()
	maxRecords, maxBytes := h.store.Limits()
	writeJSON(w, http.StatusOK, info{
		Ready:         true,
		Identity:      h.id,
		WriteKeys:     []string{},
		RecordVersion: store.RecordVersion,
		Cursor:        view.Cursor,
		Store: infoStore{
			storeStats: storeStats{Requests: counts.Requests, Events: counts.Events, Control: counts.Control},
			Bytes:      stats.Bytes, Evicted: stats.Evicted, EvictedThrough: stats.EvictedThrough,
			MaxRequests: maxRecords, MaxBytes: maxBytes,
		},
	})
}

type resetResult struct {
	Cursor  uint64     `json:"cursor"`
	Removed storeStats `json:"removed"`
}

func (h *Handler) reset(w http.ResponseWriter, _ *http.Request) {
	counts := countRecords(h.store.Reset())
	writeJSON(w, http.StatusOK, resetResult{
		Cursor:  h.store.Cursor(),
		Removed: storeStats{Requests: counts.Requests, Events: counts.Events, Control: counts.Control},
	})
}

// shutdown answers 202 and then calls the hook, so the caller gets the
// answer before the server drains. A non-loopback bind refuses it.
func (h *Handler) shutdown(w http.ResponseWriter, _ *http.Request) {
	if !IsLoopback(h.id.Bind) {
		writeError(w, apiError{status: http.StatusForbidden, Code: "stop_disabled",
			Message: "shutdown is refused on a non-loopback bind; stop the process or container instead",
			Next:    strp("rudder-cli dev listen --help")})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"stopping": true, "serverId": h.id.ServerID})
	if h.cfg.Shutdown != nil {
		go h.cfg.Shutdown()
	}
}

// IsLoopback reports whether a bind address only accepts local
// connections. It is the one loopback rule of the server and the CLI.
func IsLoopback(bind string) bool {
	if bind == "localhost" {
		return true
	}
	ip := net.ParseIP(bind)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeRaw writes an already encoded JSON body.
func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
