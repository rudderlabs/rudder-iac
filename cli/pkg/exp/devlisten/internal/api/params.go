package api

import (
	"net/url"
	"slices"
	"strconv"
	"time"
)

const (
	maxWait         = 110 * time.Second
	defaultLimit    = 100
	defaultMaxBytes = 24000
)

// Param describes one query parameter. The CLI golden test reads these
// tables, so a flag and its parameter cannot drift apart.
type Param struct {
	Name       string
	Default    string
	Repeatable bool
}

var (
	pSince      = Param{Name: "since", Default: "0"}
	pServerID   = Param{Name: "serverId"}
	pLimit      = Param{Name: "limit", Default: strconv.Itoa(defaultLimit)}
	pOrder      = Param{Name: "order", Default: "asc"}
	pFields     = Param{Name: "fields", Repeatable: true}
	pMaxBytes   = Param{Name: "maxBytes", Default: strconv.Itoa(defaultMaxBytes)}
	pRoute      = Param{Name: "route", Repeatable: true}
	pStatusCode = Param{Name: "statusCode", Repeatable: true}
	pWait       = Param{Name: "wait", Default: "0s"}
	pMin        = Param{Name: "min", Default: "1"}
)

// Params lists the parameters of each query route, keyed by route name.
var Params = map[string][]Param{
	"events": {
		pSince, pServerID, pLimit, pOrder,
		{Name: "view", Default: "compact"}, pFields, {Name: "include", Repeatable: true}, pMaxBytes,
		{Name: "event", Repeatable: true}, {Name: "type", Repeatable: true}, pRoute, pStatusCode,
		{Name: "userId"}, {Name: "anonymousId"}, pWait, pMin,
	},
	"requests": {
		pSince, pServerID, pLimit, pOrder,
		{Name: "kind", Default: "ingestion"}, pRoute, pStatusCode, {Name: "failed"}, {Name: "stage"},
		pFields, pMaxBytes, pWait, pMin,
	},
	"requests/{seq}": {pServerID, pFields, pMaxBytes},
	"summary":        {pSince, pServerID},
	"info":           {},
}

// params reads query parameters under the contract section 4.2 rules: an
// unknown parameter is an error, and a singleton given twice is an error.
type params struct {
	values url.Values
	route  string
	err    *apiError
}

func parseParams(values url.Values, route string) (*params, *apiError) {
	known := Params[route]
	for name := range values {
		i := slices.IndexFunc(known, func(p Param) bool { return p.Name == name })
		if i < 0 {
			return nil, &apiError{status: 400, Code: "unknown_parameter", Message: "unknown parameter: " + name,
				Param: &name, Next: helpNext(route)}
		}
		if !known[i].Repeatable && len(values[name]) > 1 {
			return nil, invalidParam(route, name, "repeated")
		}
	}
	return &params{values: values, route: route}, nil
}

func (p *params) fail(name, format string, args ...any) {
	if p.err == nil {
		p.err = invalidParam(p.route, name, format, args...)
	}
}

// failWith records an error whose next is a corrected command.
func (p *params) failWith(name, next string, details map[string]any, format string, args ...any) {
	if p.err == nil {
		p.err = invalidParam(p.route, name, format, args...)
		p.err.Next = strp(next)
		p.err.Details = details
	}
}

// single returns the value of a singleton parameter, or "".
func (p *params) single(name string) string { return p.values.Get(name) }

func (p *params) list(name string) []string { return p.values[name] }

func (p *params) uint(name string, def uint64) uint64 {
	raw := p.single(name)
	if raw == "" {
		return def
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		p.fail(name, "%q is not a non-negative integer", raw)
	}
	return n
}

func (p *params) intIn(name string, def, lo, hi int) int {
	raw := p.single(name)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		p.fail(name, "%q is not an integer from %d to %d", raw, lo, hi)
	}
	return n
}

func (p *params) ints(name string) []int {
	var out []int
	for _, raw := range p.list(name) {
		n, err := strconv.Atoi(raw)
		if err != nil {
			p.fail(name, "%q is not an integer", raw)
		}
		out = append(out, n)
	}
	return out
}

// oneOf returns the value when it is in allowed, else def when absent.
func (p *params) oneOf(name, def string, allowed ...string) string {
	raw := p.single(name)
	if raw == "" {
		return def
	}
	if !slices.Contains(allowed, raw) {
		p.fail(name, "%q is not one of %v", raw, allowed)
	}
	return raw
}

// optBool is nil when the parameter is absent: a boolean filter only
// applies when sent.
func (p *params) optBool(name string) *bool {
	raw := p.single(name)
	if raw == "" {
		return nil
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		p.fail(name, "%q is not true or false", raw)
	}
	return &b
}

func (p *params) wait() time.Duration {
	raw := p.single("wait")
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	switch {
	case err != nil || d < 0:
		p.fail("wait", "%q is not a Go duration such as 30s", raw)
	case d > maxWait:
		p.fail("wait", "%s exceeds the maximum of 110s", raw)
	}
	return d
}

// order accepts asc only; desc is a later phase (contract 4.2).
func (p *params) order() {
	p.oneOf("order", "asc", "asc")
}

// maxBytes returns the page cap; 0 turns it off.
func (p *params) maxBytes() int {
	return p.intIn("maxBytes", defaultMaxBytes, 0, 1<<30)
}
