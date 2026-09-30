package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// summary is the counts block of every /events envelope. Request-level
// filters (since, statusCode, writeKey) narrow every count; event-level
// filters (event, type, userId, anonymousId) narrow the event counts. Each
// event name and write key the caller filtered on is listed, at 0 when
// nothing arrived, so a `jq -e` check sees absence without an assertion
// flag.
type summary struct {
	Requests requestCounts  `json:"requests"`
	Events   eventCounts    `json:"events"`
	ByEvent  map[string]int `json:"byEvent"`
	// ByWriteKey is keyed by the stored, masked form of each key.
	ByWriteKey map[string]*keyCounts `json:"byWriteKey"`
	Control    controlCounts         `json:"control"`
	BySource   sourceCounts          `json:"bySource"`
	Diagnosis  []diagnosisItem       `json:"diagnosis"`
}

type requestCounts struct {
	Total        int            `json:"total"`
	Failed       int            `json:"failed"`
	ByRoute      map[string]int `json:"byRoute"`
	ByStatusCode map[string]int `json:"byStatusCode"`
	ByOutcome    map[string]int `json:"byOutcome"`
	ByStage      map[string]int `json:"byStage"`
}

type eventCounts struct {
	Total  int            `json:"total"`
	ByType map[string]int `json:"byType"`
	// byEvent is lifted to the summary level.
	byEvent map[string]int
}

type controlCounts struct {
	Total              int `json:"total"`
	SourceConfig       int `json:"sourceConfig"`
	SourceConfigFailed int `json:"sourceConfigFailed"`
	Preflight          int `json:"preflight"`
	PluginPath         int `json:"pluginPath"`
	Other              int `json:"other"`
}

type diagnosisItem struct {
	Code    string `json:"code"`
	Count   int    `json:"count"`
	Message string `json:"message"`
	Next    string `json:"next"`
}

// summarize counts the window for the summary block.
func (h *Handler) summarize(records []store.Record, q eventsQuery) summary {
	c := countSummary(records, q)
	for _, name := range q.event {
		if !strings.HasSuffix(name, "*") {
			c.events.byEvent[name] += 0
		}
	}
	for _, key := range q.writeKey {
		c.keys.of(store.MaskWriteKey(key))
	}
	return summary{
		Requests: c.requests, Events: c.events, ByEvent: c.events.byEvent, ByWriteKey: c.keys, Control: c.control,
		BySource: c.sources, Diagnosis: h.diagnose(c, q),
	}
}

type counts struct {
	requests requestCounts
	events   eventCounts
	control  controlCounts
	sources  sourceCounts
	keys     keyTable
}

// keyCounts are the ingestion requests and matching events of one key.
type keyCounts struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
}

type keyTable map[string]*keyCounts

func (t keyTable) of(key string) *keyCounts {
	if t[key] == nil {
		t[key] = &keyCounts{}
	}
	return t[key]
}

// countSummary counts the records that pass the request filters, and in
// events only the events that pass the event filters.
func countSummary(records []store.Record, q eventsQuery) counts {
	c := counts{
		requests: requestCounts{ByRoute: map[string]int{}, ByStatusCode: map[string]int{}, ByOutcome: map[string]int{},
			ByStage: map[string]int{}},
		events:  eventCounts{ByType: map[string]int{}, byEvent: map[string]int{}},
		sources: newSourceCounts(),
		keys:    keyTable{},
	}
	for _, rec := range records {
		if !q.matchesRecord(rec) {
			continue
		}
		c.sources.add(rec)
		if rec.Kind == "control" {
			c.control.add(rec)
			continue
		}
		c.requests.add(rec)
		matched := c.events.add(rec, q.matchesEvent)
		k := c.keys.of(rec.WriteKey)
		k.Requests++
		k.Events += matched
	}
	return c
}

func (c *requestCounts) add(rec store.Record) {
	c.Total++
	if rec.Failed {
		c.Failed++
	}
	c.ByRoute[rec.Route]++
	c.ByStatusCode[strconv.Itoa(rec.StatusCode)]++
	c.ByOutcome[rec.Outcome]++
	if rec.Rejection != nil {
		c.ByStage[rec.Rejection.Stage]++
	}
}

// add counts the events of rec that match and returns how many did.
func (c *eventCounts) add(rec store.Record, match func(store.Event) bool) int {
	n := 0
	for _, ev := range rec.Events {
		if !match(ev) {
			continue
		}
		n++
		c.Total++
		if ev.Type != nil {
			c.ByType[*ev.Type]++
		}
		if ev.Event != nil {
			c.byEvent[*ev.Event]++
		}
	}
	return n
}

func (c *controlCounts) add(rec store.Record) {
	c.Total++
	switch {
	case rec.Request.Method == http.MethodOptions:
		c.Preflight++
	case rec.Route == "/sourceConfig":
		c.SourceConfig++
		if rec.Failed {
			c.SourceConfigFailed++
		}
	default:
		c.Other++
	}
}

func sumStages(byStage map[string]int, stages []string) int {
	n := 0
	for _, stage := range stages {
		n += byStage[stage]
	}
	return n
}

var bodyStages = []string{"decode", "body", "parse", "batch", "identity", "size"}

// diagnose applies the contract section 4.7 rules in order. Every rule that
// holds appears. A next never carries a captured value.
func (h *Handler) diagnose(c counts, q eventsQuery) []diagnosisItem {
	var d diagnosis
	d.transport(c.requests, c.control, h.curlTrack(), q.since)
	d.browser(c.sources, c.control, q.since)
	d.rejections(c.requests, q.since)
	d.missingKey(c.keys, q.since)
	d.add(c.requests.Total > 0 && c.requests.Failed == 0, "all_accepted", c.requests.Total,
		"Every received request was accepted. This says nothing about events that never arrived: "+
			"compare summary.byEvent with the events you expect, then list them.",
		q.filterArgs(newCommand(routeCommands["events"]).num("since", q.since)).flag("json").String())
	return d.items
}

// curlTrack is a shell command that sends one test track to this server.
func (h *Handler) curlTrack() string {
	return "curl -fsS -u " + shellQuote(h.id.WriteKey+":") + " -H 'Content-Type: application/json' " + // gitleaks:allow the listener key "dev" is not a secret
		`-d '{"event":"dev check","userId":"dev"}' ` + shellQuote(h.id.URL+"/v1/track")
}

type diagnosis struct{ items []diagnosisItem }

func (d *diagnosis) add(ok bool, code string, count int, message, next string) {
	if d.items == nil {
		d.items = []diagnosisItem{}
	}
	if ok {
		d.items = append(d.items, diagnosisItem{Code: code, Count: count, Message: message, Next: next})
	}
}

// transport covers requests that never became events.

// requestsCmd is dev requests list after the caller's cursor.
func requestsCmd(since uint64) *command {
	return newCommand(routeCommands["requests"]).num("since", since)
}

// transport covers requests that never became events.
func (d *diagnosis) transport(req requestCounts, ctl controlCounts, curlTrack string, since uint64) {
	d.add(req.Total == 0 && ctl.Total == 0, "nothing_received", 0,
		"No request reached the listener. Send a test track with next; if it arrives, check the SDK "+
			"dataPlaneUrl, configUrl and write key (app checklist in rudder-cli dev listen --help).",
		curlTrack)
	d.add(ctl.Preflight > 0 && ctl.SourceConfig == 0 && req.Total == 0, "preflight_only", ctl.Preflight,
		"Only CORS preflights arrived. Read response.headers of the control requests.",
		requestsCmd(since).bare("kind", "control").flag("json").String())
	d.add(ctl.SourceConfigFailed > 0, "sdk_config_rejected", ctl.SourceConfigFailed,
		"The SDK could not load its configuration from /sourceConfig.",
		requestsCmd(since).bare("kind", "control").flag("failed").flag("json").String())
	d.add(ctl.SourceConfig > 0 && ctl.SourceConfigFailed == 0 && req.Total == 0, "sdk_loaded_no_events",
		ctl.SourceConfig,
		"The SDK loaded its configuration but sent no events. Check that the app calls track or page, and that the SDK does not wait on CDN plugins.",
		requestsCmd(since).bare("kind", "control").flag("json").String())
}

// browser covers a browser SDK that never reached the listener while a
// server SDK did.
func (d *diagnosis) browser(src sourceCounts, ctl controlCounts, since uint64) {
	d.add(src.sdkRequests() > 0 && src.browser().Requests == 0 && ctl.SourceConfig == 0 && ctl.Preflight == 0,
		"no_browser_traffic", src.sdkRequests(),
		"Server SDK requests arrived, but no browser request, no /sourceConfig and no preflight. "+
			"If the app also runs a browser SDK, check in order: 1. its configUrl is the listener URL; "+
			"2. its dataPlaneUrl is the listener URL; 3. the app's analytics gate (consent, env flag) is on.",
		requestsCmd(since).bare("kind", "control").flag("json").String())
}

func (d *diagnosis) missingKey(keys keyTable, since uint64) {
	missing := keys[""]
	if missing == nil || missing.Requests == 0 {
		return
	}
	d.add(true, "missing_write_key", missing.Requests,
		strconv.Itoa(missing.Requests)+" requests had no write key; RudderStack rejects these with 401.",
		newCommand(routeCommands["requests"]).num("since", since).quoted("write-key", "").flag("json").String())
}

func (d *diagnosis) rejections(req requestCounts, since uint64) {
	d.add(req.ByStage["auth"] > 0, "auth_rejected", req.ByStage["auth"],
		"Requests were rejected at the auth stage. Compare their writeKey with the SDK configuration.",
		requestsCmd(since).bare("status-code", "401").flag("json").String())
	bodyRejected := sumStages(req.ByStage, bodyStages)
	d.add(bodyRejected > 0, "body_rejected", bodyRejected,
		"Requests were rejected because of their body. Read rejection.reason on each.",
		requestsCmd(since).flag("failed").bare("status-code", "400").bare("status-code", "413").flag("json").String())
}
