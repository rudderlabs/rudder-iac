// Package api serves the query API under /_dev/v1/ (contract section 4).
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

const (
	APIVersion = "v1"
	Prefix     = "/_dev/"
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

type Handler struct {
	store *store.Store
	id    Identity
	now   func() time.Time
}

func New(st *store.Store, id Identity) *Handler {
	return &Handler{store: st, id: id, now: time.Now}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "none" && site != "same-origin" {
		writeError(w, apiError{status: http.StatusForbidden, Code: "browser_origin",
			Message: "the query API refuses cross-site browser requests"})
		return
	}
	select {
	case <-h.store.Done():
		writeError(w, errShuttingDown)
		return
	default:
	}

	routes := map[string]http.HandlerFunc{
		"/_dev/v1/info":     h.info,
		"/_dev/v1/events":   h.events,
		"/_dev/v1/requests": h.requests,
	}
	handle, ok := routes[r.URL.Path]
	switch {
	case !ok:
		writeError(w, apiError{status: http.StatusNotFound, Code: "not_found", Message: "no route " + r.URL.Path})
	case r.Method != http.MethodGet:
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, apiError{status: http.StatusMethodNotAllowed, Code: "method_not_allowed",
			Message: r.Method + " is not allowed on " + r.URL.Path})
	default:
		handle(w, r)
	}
}

type info struct {
	Ready bool `json:"ready"`
	Identity
	WriteKeys     []string   `json:"writeKeys"`
	RecordVersion int        `json:"recordVersion"`
	Cursor        uint64     `json:"cursor"`
	Store         storeStats `json:"store"`
}

type storeStats struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
	Control  int `json:"control"`
}

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	if _, err := parseParams(r.URL.Query()); err != nil {
		writeError(w, *err)
		return
	}
	view := h.store.Since(0)
	counts := countRecords(view.Records)
	writeJSON(w, http.StatusOK, info{
		Ready:         true,
		Identity:      h.id,
		WriteKeys:     []string{},
		RecordVersion: store.RecordVersion,
		Cursor:        view.Cursor,
		Store:         storeStats{Requests: counts.Requests, Events: counts.Events, Control: counts.Control},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
