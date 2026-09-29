package api

import (
	"net/url"
	"strconv"
	"time"
)

const maxWait = 110 * time.Second

// params reads query parameters under the contract section 4.2 rules: an
// unknown parameter is an error, and a singleton given twice is an error.
type params struct {
	values url.Values
	err    *apiError
}

func parseParams(values url.Values, allowed ...string) (*params, *apiError) {
	known := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		known[name] = true
	}
	for name := range values {
		if !known[name] {
			return nil, &apiError{status: 400, Code: "unknown_parameter", Message: "unknown parameter: " + name, Param: &name}
		}
	}
	return &params{values: values}, nil
}

// single returns the one value of a singleton parameter, or "".
func (p *params) single(name string) string {
	vs := p.values[name]
	if len(vs) > 1 && p.err == nil {
		p.err = invalidParam(name, "repeated")
	}
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}

func (p *params) list(name string) []string { return p.values[name] }

func (p *params) uint(name string, def uint64) uint64 {
	raw := p.single(name)
	if raw == "" {
		return def
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil && p.err == nil {
		p.err = invalidParam(name, "%q is not a non-negative integer", raw)
	}
	return n
}

func (p *params) intIn(name string, def, lo, hi int) int {
	raw := p.single(name)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if (err != nil || n < lo || n > hi) && p.err == nil {
		p.err = invalidParam(name, "%q is not an integer from %d to %d", raw, lo, hi)
	}
	return n
}

func (p *params) wait() time.Duration {
	raw := p.single("wait")
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	switch {
	case p.err != nil:
	case err != nil || d < 0:
		p.err = invalidParam("wait", "%q is not a Go duration such as 30s", raw)
	case d > maxWait:
		p.err = invalidParam("wait", "%s exceeds the maximum of %s", raw, maxWait)
	}
	return d
}
