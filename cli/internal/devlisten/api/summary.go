package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
)

type summaryObject struct {
	APIVersion     string `json:"apiVersion"`
	ServerID       string `json:"serverId"`
	Since          uint64 `json:"since"`
	Cursor         uint64 `json:"cursor"`
	EvictedThrough uint64 `json:"evictedThrough"`
	TimedOut       bool   `json:"timedOut"`
	WaitedMs       int64  `json:"waitedMs"`
	Summary        counts `json:"summary"`
	Next           string `json:"next"`
}

type counts struct {
	Requests struct {
		Total        int            `json:"total"`
		Failed       int            `json:"failed"`
		ByRoute      map[string]int `json:"byRoute"`
		ByStatusCode map[string]int `json:"byStatusCode"`
		ByOutcome    map[string]int `json:"byOutcome"`
		ByStage      map[string]int `json:"byStage"`
	} `json:"requests"`
	Events struct {
		Total  int            `json:"total"`
		ByType map[string]int `json:"byType"`
	} `json:"events"`
	ByEvent  map[string]int `json:"byEvent"`
	Rejected struct {
		Events  int            `json:"events"`
		ByEvent map[string]int `json:"byEvent"`
	} `json:"rejected"`
	ByWriteKey map[string]*keyCounts `json:"byWriteKey"`
	Control    struct {
		Total              int `json:"total"`
		SourceConfig       int `json:"sourceConfig"`
		SourceConfigFailed int `json:"sourceConfigFailed"`
		Preflight          int `json:"preflight"`
		PluginPath         int `json:"pluginPath"`
		Other              int `json:"other"`
	} `json:"control"`
	BySource struct {
		ByChannel map[string]int        `json:"byChannel"`
		BySdk     map[string]*sdkCounts `json:"bySdk"`
	} `json:"bySource"`
	Diagnosis []diagnosisItem `json:"diagnosis"`
}

type keyCounts struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
}

type sdkCounts struct {
	Requests int `json:"requests"`
	Control  int `json:"control"`
}

type diagnosisItem struct {
	Code    string `json:"code"`
	Count   int    `json:"count"`
	Message string `json:"message"`
	Next    string `json:"next"`
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request, q *eventsQuery) {
	rd := h.read(q.since)
	var waited time.Duration
	if q.wait > 0 {
		// Start the clock before the deadline, so a wait that ran out never
		// reports less than it was given.
		began := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), q.wait)
		_, err := h.wait(ctx, rd.since, q.min, q.acceptedMatches(rd.window))
		cancel()
		waited = time.Since(began)
		switch {
		case errors.Is(err, ErrShuttingDown):
			h.fail(w, http.StatusServiceUnavailable, "shutting_down", "The listener is shutting down.",
				"rudder-cli sources listen --help")
			return
		case r.Context().Err() != nil:
			return
		}
		rd = h.reread(rd.window)
	}

	filtered := tally(rd.records, q)
	filtered.Diagnosis = h.diagnose(rd, tally(rd.records, &eventsQuery{}), filtered, q)
	writeJSON(w, http.StatusOK, summaryObject{
		APIVersion:     h.cfg.Identity.APIVersion,
		ServerID:       h.cfg.Identity.ServerID,
		Since:          rd.since,
		Cursor:         rd.cursor,
		EvictedThrough: rd.stats.EvictedThrough,
		TimedOut:       q.wait > 0 && filtered.Events.Total < q.min,
		WaitedMs:       waited.Milliseconds(),
		Summary:        filtered,
		Next:           "rudder-cli sources events --since " + strconv.FormatUint(rd.cursor, 10) + q.args() + " --json",
	})
}

// tally counts the records the filters select. A filter on the events keeps
// only the requests that carry a matching event, so control requests drop out.
func tally(records []*store.Record, q *eventsQuery) counts {
	var c counts
	c.Requests.ByRoute, c.Requests.ByStatusCode = map[string]int{}, map[string]int{}
	c.Requests.ByOutcome, c.Requests.ByStage = map[string]int{}, map[string]int{}
	c.Events.ByType, c.ByEvent, c.Rejected.ByEvent = map[string]int{}, map[string]int{}, map[string]int{}
	c.ByWriteKey, c.BySource.ByChannel, c.BySource.BySdk = map[string]*keyCounts{}, map[string]int{}, map[string]*sdkCounts{}
	c.Diagnosis = []diagnosisItem{}

	// A name or key the caller asked for shows with 0, so a check reads one key.
	for _, name := range q.events {
		if !strings.HasSuffix(name, "*") {
			c.ByEvent[name] = 0
		}
	}
	for _, key := range q.writeKeys {
		c.ByWriteKey[store.MaskWriteKey(key).Key] = &keyCounts{}
	}

	for _, r := range records {
		if !keyMatches(q.writeKeys, r) {
			continue
		}
		if r.Kind != "ingestion" {
			if !q.filtered() {
				c.addControl(r)
			}
			continue
		}
		matched := q.matches(r)
		if q.filtered() && len(matched) == 0 {
			continue
		}
		c.addRequest(r, matched)
	}
	return c
}

func (c *counts) addRequest(r *store.Record, matched []int) {
	o := outcome(r)
	c.Requests.Total++
	if o != outcomeAccepted {
		c.Requests.Failed++
	}
	c.Requests.ByRoute[r.Route]++
	c.Requests.ByStatusCode[strconv.Itoa(r.Response.StatusCode)]++
	c.Requests.ByOutcome[o]++
	if r.Rejection != nil {
		c.Requests.ByStage[r.Rejection.Stage]++
	}
	key := c.ByWriteKey[r.WriteKey.Key]
	if key == nil {
		key = &keyCounts{}
		c.ByWriteKey[r.WriteKey.Key] = key
	}
	key.Requests++
	c.sdk(r).Requests++
	if len(r.Events) > 0 {
		if channel := readEvent(r.Events[0].Message, r.Route).channel; channel != "" {
			c.BySource.ByChannel[channel]++
		}
	}

	for _, i := range matched {
		k := readEvent(r.Events[i].Message, r.Route)
		if o != outcomeAccepted {
			c.Rejected.Events++
			if k.event != "" {
				c.Rejected.ByEvent[k.event]++
			}
			continue
		}
		c.Events.Total++
		key.Events++
		if k.typ != "" {
			c.Events.ByType[k.typ]++
		}
		if k.event != "" {
			c.ByEvent[k.event]++
		}
	}
}

func (c *counts) addControl(r *store.Record) {
	c.Control.Total++
	c.sdk(r).Control++
	switch {
	case r.Request.Method == http.MethodOptions:
		c.Control.Preflight++
	case r.Route == "/sourceConfig" || r.Route == "/sourceConfig/":
		c.Control.SourceConfig++
		if r.Response.StatusCode < 200 || r.Response.StatusCode > 299 {
			c.Control.SourceConfigFailed++
		}
	case strings.Contains(r.Route, "plugins"):
		c.Control.PluginPath++
	default:
		c.Control.Other++
	}
}

func (c *counts) sdk(r *store.Record) *sdkCounts {
	class := sdkClass(r)
	s := c.BySource.BySdk[class]
	if s == nil {
		s = &sdkCounts{}
		c.BySource.BySdk[class] = s
	}
	return s
}

// sdkClass guesses the SDK family from the User-Agent and the library name
// of the first event. Every browser sends a Mozilla User-Agent, beacons too.
func sdkClass(r *store.Record) string {
	ua := strings.ToLower(r.Request.Headers["User-Agent"])
	library := ""
	if len(r.Events) > 0 {
		library = strings.ToLower(readEvent(r.Events[0].Message, r.Route).library)
	}
	has := func(s string, parts ...string) bool {
		for _, p := range parts {
			if strings.Contains(s, p) {
				return true
			}
		}
		return false
	}
	switch {
	case strings.HasPrefix(ua, "mozilla/") || has(library, "javascript"):
		return "browser"
	case has(library, "android", "ios", "flutter", "react-native", "unity", "kotlin", "swift") ||
		has(ua, "okhttp", "dalvik", "cfnetwork"):
		return "mobile"
	case has(library, "node") || strings.HasPrefix(ua, "node") || strings.HasPrefix(ua, "axios/"):
		return "node"
	case strings.HasSuffix(library, "-go") || strings.HasPrefix(ua, "go-http-client"):
		return "go"
	}
	return "other"
}

// bodyStages are the stages that read and parse a request body.
var bodyStages = []string{"decode", "body", "parse", "batch", "identity", "size"}

// diagnose reads the whole window, not the filters: a typo in a filter must
// not read as a listener that received nothing. Every rule that holds is
// listed, in the order of what it rules out first. When the window lost
// requests to the store budget, no rule can vouch for what arrived.
func (h *Handler) diagnose(rd reading, all, filtered counts, q *eventsQuery) []diagnosisItem {
	var (
		out   = []diagnosisItem{}
		win   = rd.window
		stats = rd.stats
		since = strconv.FormatUint(win.since, 10)
		add   = func(code string, count int, message, next string) {
			out = append(out, diagnosisItem{code, count, message, next})
		}
		reqs    = all.Requests.Total
		control = all.Control
	)
	requests := func(query string) string { return h.curl("requests?since=" + since + query) }
	if rd.lost {
		out = append(out, evicted(rd, q))
	}

	// The store counts ingestion requests only, so control requests alone
	// still read as nothing received. Eviction needs requests to evict.
	received := stats.Requests > 0 || stats.EvictedThrough > 0
	if !received {
		key := "dev"
		if len(h.cfg.WriteKeys) > 0 {
			key = "KEY"
		}
		add("nothing_received", 0,
			"The listener has received no event request since it started. Point the SDK at this URL, or send the check request.",
			"curl -fsS -u '"+key+":' -H 'Content-Type: application/json' -d '{\"event\":\"dev check\",\"userId\":\"dev\"}' '"+
				h.cfg.Identity.URL+"/v1/track'")
	}
	// An empty window that lost requests is not empty because nothing came,
	// so the rules below that read an empty window skip it.
	if received && reqs == 0 && !rd.lost {
		add("nothing_new", 0,
			fmt.Sprintf("No request after cursor %d; the listener holds %d. The app sent before the cursor, or has not sent yet.",
				win.since, stats.Requests),
			"rudder-cli sources events --since 0 --json")
	}
	if control.Preflight > 0 && control.SourceConfig == 0 && reqs == 0 && !rd.lost {
		add("preflight_only", control.Preflight,
			"The browser sent CORS preflights only, so it never sent the requests that carry events.",
			requests("&kind=control"))
	}
	if control.SourceConfigFailed > 0 {
		add("sdk_config_rejected", control.SourceConfigFailed,
			"The SDK asked for its config and got 401 or 400: its write key is not on the --write-key allowlist.",
			requests("&kind=control&failed=true"))
	}
	if control.SourceConfig > 0 && control.SourceConfigFailed == 0 && reqs == 0 && !rd.lost {
		add("sdk_loaded_no_events", control.SourceConfig,
			"The SDK loaded its config but sent no event: check the SDK plugins, or the action never ran.",
			requests("&kind=control"))
	}
	if browser := all.BySource.BySdk["browser"]; reqs > 0 && (browser == nil || browser.Requests == 0) &&
		control.SourceConfig == 0 && control.Preflight == 0 {
		add("no_browser_traffic", reqs,
			"If the app also runs a browser SDK, the browser did not reach the listener: no browser request, config request or preflight arrived.",
			"rudder-cli sources listen --help")
	}
	if n := all.Requests.ByStage["auth"]; n > 0 {
		add("auth_rejected", n,
			fmt.Sprintf("%s a missing write key or one that is not on the --write-key allowlist.", requestsVerb(n, "had", "had")),
			requests("&statusCode=401&view=compact"))
	}
	body := 0
	for _, stage := range bodyStages {
		body += all.Requests.ByStage[stage]
	}
	if body > 0 {
		add("body_rejected", body,
			requestsVerb(body, "was", "were")+" rejected while the body was read: bad gzip, empty body, invalid JSON, "+
				"wrong batch shape, no identity, or too large. Read rejection.reason.",
			requests("&failed=true&view=compact"))
	}
	// Under an allowlist a missing key is an auth rejection instead.
	if missing := all.ByWriteKey[""]; missing != nil && len(h.cfg.WriteKeys) == 0 {
		add("missing_write_key", missing.Requests,
			requestsVerb(missing.Requests, "arrived", "arrived")+" with no write key. The listener accepts them; RudderStack answers 401.",
			requests("&writeKey="))
	}
	if reqs > 0 && all.Requests.Failed == 0 && !rd.lost {
		add("all_accepted", reqs,
			"Every request that arrived was accepted. This says nothing about events that never came: read byEvent.",
			"rudder-cli sources events --since "+since+q.args()+" --json")
	}
	if (q.filtered() || len(q.writeKeys) > 0) && reqs > 0 && filtered.Requests.Total == 0 && filtered.Control.Total == 0 {
		add("filtered_empty", reqs,
			"Requests arrived, but the filters match none. Names are exact and case-sensitive.",
			"rudder-cli sources events --since "+since+" --json")
	}
	return out
}

// CodeEvicted marks a window that lost requests to the store budget, in the
// diagnosis and in the stream headers.
const CodeEvicted = "events_evicted"

// evicted describes the loss. A cursor window counts the lost requests,
// because each request takes the next seq; a time window cannot.
func evicted(rd reading, q *eventsQuery) diagnosisItem {
	count, subject := 0, "Requests were"
	if rd.floor.IsZero() {
		count = int(rd.stats.EvictedThrough - rd.since)
		subject = requestsVerb(count, "was", "were")
	}
	return diagnosisItem{
		Code:  CodeEvicted,
		Count: count,
		Message: fmt.Sprintf("%s evicted from this window: the listener keeps the last %d requests or %d MiB. "+
			"The counts and the events are incomplete.", subject, rd.stats.MaxRequests, rd.stats.MaxBytes>>20),
		// The window after the newest evicted request is complete.
		Next: "rudder-cli sources events --since " + strconv.FormatUint(rd.stats.EvictedThrough, 10) + q.args() + " --json",
	}
}

func requestsVerb(n int, one, many string) string {
	if n == 1 {
		return "1 request " + one
	}
	return strconv.Itoa(n) + " requests " + many
}

// args repeats the caller's filters as flags, so a next command reads the
// same selection.
func (q *eventsQuery) args() string {
	var b strings.Builder
	flag := func(name, value string) { b.WriteString(" --" + name + " " + shellWord(value)) }
	for _, v := range q.events {
		flag("event", v)
	}
	for _, v := range q.types {
		flag("type", v)
	}
	if q.userID != nil {
		flag("user-id", *q.userID)
	}
	if q.anonymousID != nil {
		flag("anonymous-id", *q.anonymousID)
	}
	for _, v := range q.writeKeys {
		flag("write-key", v)
	}
	if q.serverID != "" {
		flag("server-id", q.serverID)
	}
	return b.String()
}

var plainWord = regexp.MustCompile(`^[A-Za-z0-9._/:@%+=,-]+$`)

// shellWord quotes a value for a POSIX shell, so a next command runs as
// printed whatever the caller typed.
func shellWord(s string) string {
	if plainWord.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
