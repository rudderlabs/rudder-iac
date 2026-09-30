package api

import (
	"net/http"
	"strconv"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

type summary struct {
	APIVersion string          `json:"apiVersion"`
	ServerID   string          `json:"serverId"`
	Since      uint64          `json:"since"`
	Cursor     uint64          `json:"cursor"`
	Requests   requestCounts   `json:"requests"`
	Events     eventCounts     `json:"events"`
	Control    controlCounts   `json:"control"`
	BySource   sourceCounts    `json:"bySource"`
	Expected   []expectation   `json:"expected,omitempty"`
	Diagnosis  []diagnosisItem `json:"diagnosis"`
}

type requestCounts struct {
	Total        int            `json:"total"`
	Failed       int            `json:"failed"`
	Probes       int            `json:"probes"`
	ByRoute      map[string]int `json:"byRoute"`
	ByStatusCode map[string]int `json:"byStatusCode"`
	ByOutcome    map[string]int `json:"byOutcome"`
	ByStage      map[string]int `json:"byStage"`
}

type eventCounts struct {
	Total   int            `json:"total"`
	ByType  map[string]int `json:"byType"`
	ByEvent map[string]int `json:"byEvent"`
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
	Code    string   `json:"code"`
	Count   int      `json:"count"`
	Message string   `json:"message"`
	Events  []string `json:"events,omitempty"`
	Next    string   `json:"next"`
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	p, err := parseParams(r.URL.Query(), "summary")
	if err != nil {
		writeError(w, *err)
		return
	}
	since := p.uint("since", 0)
	expected := parseExpect(p)
	if p.err == nil {
		p.err = h.checkServerID(p.single("serverId"))
	}
	if p.err != nil {
		writeError(w, *p.err)
		return
	}
	view := h.store.Since(since)
	all := countSummary(view.Records, true)
	expected = checkExpected(expected, all.events.ByEvent)
	writeJSON(w, http.StatusOK, summary{
		APIVersion: APIVersion, ServerID: h.id.ServerID, Since: since, Cursor: max(since, view.Cursor),
		Requests: all.requests, Events: all.events, Control: all.control, BySource: all.sources,
		Expected: expected, Diagnosis: diagnose(countSummary(view.Records, false), expected, since),
	})
}

type counts struct {
	requests requestCounts
	events   eventCounts
	control  controlCounts
	sources  sourceCounts
}

// countSummary counts records; probes are skipped unless withProbes is set.
func countSummary(records []store.Record, withProbes bool) counts {
	c := counts{
		requests: requestCounts{ByRoute: map[string]int{}, ByStatusCode: map[string]int{}, ByOutcome: map[string]int{},
			ByStage: map[string]int{}},
		events:  eventCounts{ByType: map[string]int{}, ByEvent: map[string]int{}},
		sources: newSourceCounts(),
	}
	for _, rec := range records {
		if rec.Probe && !withProbes {
			continue
		}
		c.sources.add(rec)
		if rec.Kind == "control" {
			c.control.add(rec)
			continue
		}
		c.requests.add(rec)
		c.events.add(rec)
	}
	return c
}

func (c *requestCounts) add(rec store.Record) {
	c.Total++
	if rec.Failed {
		c.Failed++
	}
	if rec.Probe {
		c.Probes++
	}
	c.ByRoute[rec.Route]++
	c.ByStatusCode[strconv.Itoa(rec.StatusCode)]++
	c.ByOutcome[rec.Outcome]++
	if rec.Rejection != nil {
		c.ByStage[rec.Rejection.Stage]++
	}
}

func (c *eventCounts) add(rec store.Record) {
	for _, ev := range rec.Events {
		c.Total++
		if ev.Type != nil {
			c.ByType[*ev.Type]++
		}
		if ev.Event != nil {
			c.ByEvent[*ev.Event]++
		}
	}
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
func diagnose(c counts, expected []expectation, since uint64) []diagnosisItem {
	var d diagnosis
	d.transport(c.requests, c.control)
	d.browser(c.sources, c.control)
	d.rejections(c.requests)
	d.expectations(expected, since)
	d.add(c.requests.Total > 0 && c.requests.Failed == 0, "all_accepted", c.requests.Total,
		"Every request was accepted. List the events to check names and properties.",
		newCommand(routeCommands["events"]).num("since", since).bare("view", viewSummary).flag("json").String())
	return d.items
}

type diagnosis struct{ items []diagnosisItem }

func (d *diagnosis) addEvents(ok bool, code string, events []string, message, next string) {
	d.add(ok, code, len(events), message, next)
	if ok {
		d.items[len(d.items)-1].Events = events
	}
}

func (d *diagnosis) add(ok bool, code string, count int, message, next string) {
	if d.items == nil {
		d.items = []diagnosisItem{}
	}
	if ok {
		d.items = append(d.items, diagnosisItem{Code: code, Count: count, Message: message, Next: next})
	}
}

// transport covers requests that never became events.
func (d *diagnosis) transport(req requestCounts, ctl controlCounts) {
	d.add(req.Total == 0 && ctl.Total == 0, "nothing_received", 0,
		"No request reached the listener. Send a probe; if it arrives, check the SDK dataPlaneUrl, configUrl and write key.",
		"rudder-cli dev send --json")
	d.add(ctl.Preflight > 0 && ctl.SourceConfig == 0 && req.Total == 0, "preflight_only", ctl.Preflight,
		"Only CORS preflights arrived. Read response.headers of the control requests.",
		"rudder-cli dev requests list --kind control --json")
	d.add(ctl.SourceConfigFailed > 0, "sdk_config_rejected", ctl.SourceConfigFailed,
		"The SDK could not load its configuration from /sourceConfig.",
		"rudder-cli dev requests list --kind control --failed --json")
	d.add(ctl.SourceConfig > 0 && ctl.SourceConfigFailed == 0 && req.Total == 0, "sdk_loaded_no_events",
		ctl.SourceConfig,
		"The SDK loaded its configuration but sent no events. Check that the app calls track or page, and that the SDK does not wait on CDN plugins.",
		"rudder-cli dev requests list --kind control --json")
}

// browser covers a browser SDK that never reached the listener while a
// server SDK did.
func (d *diagnosis) browser(src sourceCounts, ctl controlCounts) {
	d.add(src.sdkRequests() > 0 && src.browser().Requests == 0 && ctl.SourceConfig == 0 && ctl.Preflight == 0,
		"no_browser_traffic", src.sdkRequests(),
		"Server SDK requests arrived, but no browser request, no /sourceConfig and no preflight. "+
			"If the app also runs a browser SDK, it did not load or sends elsewhere: "+
			"follow the app checklist in rudder-cli dev listen --help.",
		"rudder-cli dev requests list --kind control --json")
}

func (d *diagnosis) rejections(req requestCounts) {
	d.add(req.ByStage["auth"] > 0, "auth_rejected", req.ByStage["auth"],
		"Requests were rejected at the auth stage. Compare their writeKey with the SDK configuration.",
		"rudder-cli dev requests list --failed --stage auth --json")
	bodyRejected := sumStages(req.ByStage, bodyStages)
	d.add(bodyRejected > 0, "body_rejected", bodyRejected,
		"Requests were rejected because of their body. Read rejection.reason on each.",
		"rudder-cli dev requests list --failed --json")
}
