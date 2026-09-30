package api

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// filters are the parameters /events and /requests share.
type filters struct {
	since        uint64
	serverID     string
	limit        int
	min          int
	wait         time.Duration
	maxBytes     int
	sentMaxBytes string
	route        []string
	statusCode   []int
}

func parseFilters(p *params) filters {
	p.order()
	f := filters{
		since:        p.uint("since", 0),
		serverID:     p.single("serverId"),
		limit:        p.intIn("limit", defaultLimit, 1, 1000),
		wait:         p.wait(),
		maxBytes:     p.maxBytes(),
		sentMaxBytes: p.single("maxBytes"),
		route:        p.list("route"),
		statusCode:   p.ints("statusCode"),
	}
	f.min = p.intIn("min", 1, 1, f.limit)
	return f
}

func (f filters) matchesRecord(rec store.Record) bool {
	return (len(f.route) == 0 || slices.Contains(f.route, rec.Route)) &&
		(len(f.statusCode) == 0 || slices.Contains(f.statusCode, rec.StatusCode))
}

func (f filters) args(c *command) {
	if f.serverID != "" {
		c.quoted("server-id", f.serverID)
	}
}

func (f filters) maxBytesArg(c *command) {
	if f.sentMaxBytes != "" {
		c.quoted("max-bytes", f.sentMaxBytes)
	}
}

// group is the rendered items of one request. A page never splits one.
type group struct {
	seq      uint64
	items    []json.RawMessage
	stripped []string
}

type scanResult struct {
	groups     []group
	count      int
	hasMore    bool
	cursor     uint64
	unfiltered unfiltered
}

// scan builds one page under contract section 4.4: it never splits a
// request, and the cursor never skips or repeats a request.
func scan(view store.View, since uint64, limit int, render func(store.Record) group) scanResult {
	res := scanResult{cursor: max(since, view.Cursor), unfiltered: countRecords(view.Records)}
	for i, rec := range view.Records {
		g := render(rec)
		if len(g.items) == 0 {
			continue
		}
		if res.count >= limit {
			res.hasMore = true
			res.cursor = view.Records[i-1].Seq
			break
		}
		res.groups = append(res.groups, g)
		res.count += len(g.items)
	}
	return res
}

// poll long-polls until min matches exist or the wait ends. ok is false when
// the client left before an answer.
func (h *Handler) poll(ctx context.Context, f filters, scanFn func(store.View) scanResult) (
	res scanResult, timedOut, ok bool, err *apiError,
) {
	timer := time.NewTimer(f.wait)
	defer timer.Stop()
	for {
		view := h.store.Since(f.since)
		res = scanFn(view)
		if f.wait == 0 || res.count >= f.min {
			return res, false, true, nil
		}
		if h.waiting != nil {
			h.waiting()
		}
		switch h.waitChange(ctx, view.Changed, timer.C) {
		case waitTimedOut:
			res = scanFn(h.store.Since(f.since))
			return res, res.count < f.min, true, nil
		case waitLeft:
			return scanResult{}, false, false, nil
		case waitShutdown:
			return scanResult{}, false, false, &errShuttingDown
		}
	}
}

type waitOutcome int

const (
	waitChanged waitOutcome = iota
	waitTimedOut
	waitLeft
	waitShutdown
)

func (h *Handler) waitChange(ctx context.Context, changed <-chan struct{}, deadline <-chan time.Time) waitOutcome {
	select {
	case <-changed:
		return waitChanged
	case <-deadline:
		return waitTimedOut
	case <-ctx.Done():
		return waitLeft
	case <-h.store.Done():
		return waitShutdown
	}
}

type truncated struct {
	By           string `json:"by"`
	Kept         int    `json:"kept"`
	MatchedAfter int    `json:"matchedAfter"`
	RequestBytes int    `json:"requestBytes,omitempty"`
	Next         string `json:"next"`
}

type renderFunc func(groups []group, cursor uint64, hasMore bool, t *truncated) []byte

// fit applies maxBytes: it drops trailing whole requests until the page
// fits. When the first request alone is too large, the page is empty and
// truncated.requestBytes carries its size.
func fit(res scanResult, f filters, render renderFunc, next func(cursor uint64, oversized bool) string) []byte {
	body := render(res.groups, res.cursor, res.hasMore, nil)
	if f.maxBytes == 0 || len(body) <= f.maxBytes {
		return body
	}
	itemBytes := make([]int, len(res.groups)+1)
	for i, g := range res.groups {
		itemBytes[i+1] = itemBytes[i] + groupBytes(g)
	}
	overhead := len(body) - itemBytes[len(res.groups)]
	for n := len(res.groups) - 1; n >= 1; n-- {
		if overhead+itemBytes[n] > f.maxBytes {
			continue // a lower bound already too large
		}
		kept := countItems(res.groups[:n])
		cursor := res.groups[n-1].seq
		t := &truncated{By: "maxBytes", Kept: kept, MatchedAfter: res.count - kept, Next: next(cursor, false)}
		if body = render(res.groups[:n], cursor, true, t); len(body) <= f.maxBytes {
			return body
		}
	}
	t := &truncated{By: "maxBytes", MatchedAfter: res.count, RequestBytes: groupBytes(res.groups[0]),
		Next: next(f.since, true)}
	return render(nil, f.since, true, t)
}

func groupBytes(g group) int {
	n := 0
	for _, item := range g.items {
		n += len(item) + 1
	}
	return n
}

func countItems(groups []group) int {
	n := 0
	for _, g := range groups {
		n += len(g.items)
	}
	return n
}

func flatten(groups []group) []json.RawMessage {
	items := make([]json.RawMessage, 0, countItems(groups))
	for _, g := range groups {
		items = append(items, g.items...)
	}
	return items
}

// encode marshals without HTML escaping, so byte counts match what a
// caller reads.
func encode(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}
