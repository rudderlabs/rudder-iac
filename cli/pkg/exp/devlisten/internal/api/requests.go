package api

import (
	"encoding/json"
	"net/http"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

type requestsPage struct {
	APIVersion string           `json:"apiVersion"`
	ServerID   string           `json:"serverId"`
	Since      uint64           `json:"since"`
	Cursor     uint64           `json:"cursor"`
	HasMore    bool             `json:"hasMore"`
	Unfiltered unfiltered       `json:"unfiltered"`
	Requests   []map[string]any `json:"requests"`
}

type requestsQuery struct {
	since    uint64
	serverID string
	limit    int
	kind     string
}

func parseRequestsQuery(values map[string][]string) (requestsQuery, *apiError) {
	p, err := parseParams(values, "since", "serverId", "limit", "kind")
	if err != nil {
		return requestsQuery{}, err
	}
	q := requestsQuery{
		since:    p.uint("since", 0),
		serverID: p.single("serverId"),
		limit:    p.intIn("limit", 100, 1, 1000),
		kind:     p.single("kind"),
	}
	switch q.kind {
	case "":
		q.kind = "ingestion"
	case "ingestion", "control", "all":
	default:
		if p.err == nil {
			p.err = invalidParam("kind", "%q is not ingestion, control or all", q.kind)
		}
	}
	return q, p.err
}

// requests answers /requests: one item per request, bodies left out
// (contract section 4.3, last paragraph).
func (h *Handler) requests(w http.ResponseWriter, r *http.Request) {
	q, err := parseRequestsQuery(r.URL.Query())
	if err == nil {
		err = h.checkServerID(q.serverID)
	}
	if err != nil {
		writeError(w, *err)
		return
	}
	writeJSON(w, http.StatusOK, h.scanRequests(h.store.Since(q.since), q))
}

func (h *Handler) scanRequests(view store.View, q requestsQuery) requestsPage {
	since, limit, kind := q.since, q.limit, q.kind
	page := requestsPage{
		APIVersion: APIVersion,
		ServerID:   h.id.ServerID,
		Since:      since,
		Cursor:     max(since, view.Cursor),
		Unfiltered: countRecords(view.Records),
		Requests:   []map[string]any{},
	}
	for i, rec := range view.Records {
		if kind != "all" && rec.Kind != kind {
			continue
		}
		if len(page.Requests) == limit {
			page.HasMore = true
			page.Cursor = view.Records[i-1].Seq
			break
		}
		page.Requests = append(page.Requests, withoutBodies(rec))
	}
	return page
}

func withoutBodies(rec store.Record) map[string]any {
	raw, _ := json.Marshal(rec)
	var item map[string]any
	_ = json.Unmarshal(raw, &item)
	if req, ok := item["request"].(map[string]any); ok {
		delete(req, "body")
		delete(req, "bodyBase64")
	}
	events, _ := item["events"].([]any)
	for _, ev := range events {
		delete(ev.(map[string]any), "enrichedMessage")
	}
	return item
}
