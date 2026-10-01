// Package api serves the query API under /_dev/v1/: the part of the listener
// that the CLI read commands, the review page and curl use.
package api

import (
	"context"
	"errors"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
)

const (
	// Prefix is the path space of the query API and the review page. No
	// RudderStack ingestion path starts with "_".
	Prefix = "/_dev/"
	// UIPath is the review page.
	UIPath = "/_dev/ui/"

	base          = "/_dev/v1/"
	recordVersion = 1
)

// ErrShuttingDown ends a long-poll when the server stops.
var ErrShuttingDown = errors.New("the listener is shutting down")

// Identity is the part of the ready line that /info repeats, so a caller can
// match the two.
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

type Config struct {
	Identity Identity
	// WriteKeys is the allowlist, masked.
	WriteKeys []string
	// AllowHosts are Host names allowed besides loopback and the bind address.
	AllowHosts []string
	// Version is the CLI version.
	Version string
	// Guide is the Markdown guide that rudder-cli dev --help prints.
	Guide string
	// UI serves the review page under UIPath, after the same guards as the
	// query API.
	UI http.Handler
}

type Handler struct {
	store  *store.Store
	cfg    Config
	hosts  map[string]bool
	routes map[string]http.HandlerFunc

	// stopped ends when Stop runs; it wakes every long-poll.
	stopped context.Context
	stop    context.CancelFunc
	waiters atomic.Int64
	now     func() time.Time
}

func New(st *store.Store, cfg Config) *Handler {
	h := &Handler{store: st, cfg: cfg, hosts: map[string]bool{}, now: time.Now}
	h.stopped, h.stop = context.WithCancel(context.Background())
	for _, name := range append([]string{cfg.Identity.Bind}, cfg.AllowHosts...) {
		h.hosts[canonicalHost(name)] = true
	}
	h.routes = map[string]http.HandlerFunc{
		base:                          h.index,
		strings.TrimSuffix(base, "/"): h.index,
		base + "info":                 h.info,
		base + "events":               h.events,
		base + "requests":             h.requests,
		base + "guide":                h.guide,
	}
	return h
}

// Stop makes every later request answer 503 and wakes the long-polls, so
// the HTTP server can drain at once.
func (h *Handler) Stop() {
	h.stop()
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// The read commands trust a 200 only with this header: a dev server
	// with an HTML fallback also answers 200 on /_dev/v1/events.
	w.Header().Set("X-Dev-Server-Id", h.cfg.Identity.ServerID)

	switch {
	case !h.hostAllowed(r.Host):
		h.fail(w, http.StatusForbidden, "host_not_allowed",
			"Host "+r.Host+" is not allowed. Use a loopback address, the bind address or a name given with --allow-host.",
			"rudder-cli dev events --json")
		return
	case crossSite(r):
		h.fail(w, http.StatusForbidden, "browser_origin",
			"The query API refuses requests from other sites.", "rudder-cli dev events --json")
		return
	case h.stopped.Err() != nil:
		h.fail(w, http.StatusServiceUnavailable, "shutting_down", "The listener is shutting down.",
			"rudder-cli dev listen --help")
		return
	}

	handle, ok := h.routes[r.URL.Path]
	switch {
	case r.URL.Path == strings.TrimSuffix(UIPath, "/"):
		handle, ok = h.toUI, true
	case strings.HasPrefix(r.URL.Path, UIPath) && h.cfg.UI != nil:
		handle, ok = h.cfg.UI.ServeHTTP, true
	}
	if seq, found := strings.CutPrefix(r.URL.Path, base+"requests/"); found {
		_, err := strconv.ParseUint(seq, 10, 64)
		handle, ok = h.request, err == nil
	}
	if !ok {
		h.fail(w, http.StatusNotFound, "not_found", "No route "+r.URL.Path+".", h.curl("requests"))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		h.fail(w, http.StatusMethodNotAllowed, "method_not_allowed",
			r.Method+" is not allowed on "+r.URL.Path+".", h.curl(""))
		return
	}
	handle(w, r)
}

// hostAllowed compares the name only, so a port change or a proxy that
// rewrites the port does not lock a caller out.
func (h *Handler) hostAllowed(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	host = canonicalHost(host)
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return host != "" && h.hosts[host]
}

func canonicalHost(name string) string {
	name = strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(name, "["), "]"))
	if ip := net.ParseIP(name); ip != nil {
		return ip.String()
	}
	return name
}

// crossSite is true for a fetch from another site. A top-level navigation to
// the review page passes, so a link to it opens; the browser never hands that
// response to the other site's script.
func crossSite(r *http.Request) bool {
	site := r.Header.Get("Sec-Fetch-Site")
	if site == "" || site == "none" || site == "same-origin" {
		return false
	}
	navigation := r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, UIPath) &&
		r.Header.Get("Sec-Fetch-Mode") == "navigate" && r.Header.Get("Sec-Fetch-Dest") == "document"
	return !navigation
}

// Waiters returns the number of long-polls in progress.
func (h *Handler) Waiters() int {
	return int(h.waiters.Load())
}

// wait runs the store's long-poll until ctx ends or the server stops.
func (h *Handler) wait(ctx context.Context, since uint64, atLeast int, count func(*store.Record) int) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer context.AfterFunc(h.stopped, cancel)()
	h.waiters.Add(1)
	defer h.waiters.Add(-1)

	found, err := h.store.Wait(ctx, since, atLeast, count)
	if err != nil && h.stopped.Err() != nil {
		return found, ErrShuttingDown
	}
	return found, err
}

// noParams answers 400 when the query names any parameter, because a
// parameter the route ignores would look like a filter that matched.
// URL.Query drops a pair it cannot parse, so the raw query is parsed here.
func (h *Handler) noParams(w http.ResponseWriter, r *http.Request) bool {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.fail(w, http.StatusBadRequest, "invalid_parameter", "The query string is malformed: "+err.Error()+".", h.curl(""))
		return false
	}
	if len(query) == 0 {
		return true
	}
	name := slices.Min(slices.Collect(maps.Keys(query)))
	h.failParam(w, &paramError{code: "unknown_parameter", param: name, message: r.URL.Path + " takes no parameter " + name + "."},
		h.curl(""))
	return false
}

type endpoint struct {
	Method  string   `json:"method"`
	Path    string   `json:"path"`
	Params  []string `json:"params"`
	About   string   `json:"about"`
	Example string   `json:"example"`
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	if !h.noParams(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, struct {
		APIVersion  string            `json:"apiVersion"`
		ServerID    string            `json:"serverId"`
		Description string            `json:"description"`
		Next        string            `json:"next"`
		Curl        string            `json:"curl"`
		Links       map[string]string `json:"links"`
		Endpoints   []endpoint        `json:"endpoints"`
		Help        string            `json:"help"`
	}{
		APIVersion:  h.cfg.Identity.APIVersion,
		ServerID:    h.cfg.Identity.ServerID,
		Description: "The query API of rudder-cli dev listen. It reads what the listener captured.",
		Next:        h.curl("info"),
		Curl:        h.curl("info"),
		Links: map[string]string{
			"events": "events", "counts": "events?view=counts", "requests": "requests", "request": "requests/{seq}",
			"info": "info", "guide": "guide", "ui": UIPath,
		},
		Endpoints: []endpoint{
			{http.MethodGet, base, []string{}, "This index.", h.curl("")},
			{
				http.MethodGet, base + "events", eventsParams,
				"The accepted events as NDJSON, one per line, as the SDK sent them; the cursor is in X-Dev-Cursor. " +
					"view=counts answers the summary: counts, rejected events, a diagnosis and the cursor.",
				h.curl("events?view=counts"),
			},
			{
				http.MethodGet, base + "requests", requestsParams,
				"The request records after the cursor, refused and control requests included: status, rejection, " +
					"headers, body, response and the enrichment of each event. messageId finds the request of an event.",
				h.curl("requests?failed=true&view=compact"),
			},
			{
				http.MethodGet, base + "requests/{seq}", recordParams,
				"One request record. view=full adds each event as sent; fields=request.body&maxBytes=0 reads a large body.",
				h.recordExample(),
			},
			{http.MethodGet, base + "info", []string{}, "The listener identity, the cursor and the store counts.", h.curl("info")},
			{http.MethodGet, base + "guide", []string{}, "The guide that rudder-cli dev --help prints, as Markdown.", h.curl("guide")},
		},
		Help: "rudder-cli dev --help",
	})
}

// recordExample reads the newest record. An empty store has none, so the
// example then lists the first record.
func (h *Handler) recordExample() string {
	if seq := h.store.Cursor(); seq > 0 {
		return h.curl("requests/" + strconv.FormatUint(seq, 10))
	}
	return h.curl("requests?limit=1")
}

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	if !h.noParams(w, r) {
		return
	}
	// Cursor first: a request stored between the two reads then shows in the
	// counts, never in the cursor alone.
	cursor := h.store.Cursor()
	writeJSON(w, http.StatusOK, struct {
		Ready bool `json:"ready"`
		Identity
		WriteKeys     []string    `json:"writeKeys"`
		RecordVersion int         `json:"recordVersion"`
		Cursor        uint64      `json:"cursor"`
		Exposed       bool        `json:"exposed"`
		Version       string      `json:"version"`
		UI            string      `json:"ui"`
		Store         store.Stats `json:"store"`
	}{
		Ready:         true,
		Identity:      h.cfg.Identity,
		WriteKeys:     h.cfg.WriteKeys,
		RecordVersion: recordVersion,
		Cursor:        cursor,
		Exposed:       !IsLoopback(h.cfg.Identity.Bind),
		Version:       h.cfg.Version,
		UI:            h.cfg.Identity.URL + UIPath,
		Store:         h.store.Stats(),
	})
}

// toUI keeps the query string, so a shared view opens with its filters.
func (h *Handler) toUI(w http.ResponseWriter, r *http.Request) {
	target := UIPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (h *Handler) guide(w http.ResponseWriter, r *http.Request) {
	if !h.noParams(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, h.cfg.Guide)
}

// IsLoopback reports whether a bind address takes local connections only.
// The bind is always an IP address, because dev listen rejects names.
func IsLoopback(bind string) bool {
	ip := net.ParseIP(bind)
	return ip != nil && ip.IsLoopback()
}

// curl is a command a caller can paste; path is relative to /_dev/v1/.
func (h *Handler) curl(path string) string {
	return "curl -fsS '" + h.cfg.Identity.URL + base + path + "'"
}

type apiError struct {
	Status  int     `json:"status"`
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Param   *string `json:"param"`
	Details any     `json:"details"`
	Next    string  `json:"next"`
}

func (h *Handler) fail(w http.ResponseWriter, status int, code, message, next string) {
	writeJSON(w, status, map[string]apiError{"error": {Status: status, Code: code, Message: message, Next: next}})
}

func (h *Handler) failParam(w http.ResponseWriter, e *paramError, next string) {
	writeJSON(w, http.StatusBadRequest, map[string]apiError{"error": {
		Status: http.StatusBadRequest, Code: e.code, Message: e.message, Param: &e.param, Details: e.details, Next: next,
	}})
}

// writeJSON encodes before it writes the status, so a value it cannot
// encode answers 500 and never an empty 200.
func writeJSON(w http.ResponseWriter, status int, v any) {
	out, err := encode(v)
	if err != nil {
		status = http.StatusInternalServerError
		out, _ = encode(map[string]apiError{"error": {
			Status: status, Code: "encode_failed", Message: "The listener could not encode its answer: " + err.Error() + ".",
		}})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(out, '\n'))
}
