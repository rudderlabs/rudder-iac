package ingest

import (
	"encoding/json"
	"net/http"
)

// sourceConfig copies rudder-config-backend GET /sourceConfig
// (src/modules/source-config/routes.ts:7-24) with a minimal source: no
// destinations, so analytics-js skips integrations and reaches ready.
func (g *Gateway) sourceConfig(r *http.Request) reply {
	rep := reply{kind: "control", transport: "http", header: http.Header{}}
	if r.Method == http.MethodHead {
		rep.status = http.StatusNoContent
		return rep
	}

	rep.header.Set("Content-Type", "application/json; charset=utf-8")
	writeKey, _, ok := r.BasicAuth()
	if !ok || writeKey == "" {
		rep.status = http.StatusUnauthorized
		rep.body = []byte(`{"message":"Writekey not found in basic auth"}`)
		return rep
	}
	rep.writeKey = writeKey

	updatedAt := g.startedAt.UTC().Format("2006-01-02T15:04:05.000Z")
	body, _ := json.Marshal(map[string]any{
		"source": map[string]any{
			"id":           SourceID(writeKey),
			"name":         "rudder-cli dev listen",
			"writeKey":     writeKey,
			"enabled":      true,
			"config":       map[string]any{},
			"workspaceId":  "dev-workspace",
			"dataplanes":   map[string]any{},
			"destinations": []any{},
			"updatedAt":    updatedAt,
		},
		"updatedAt": updatedAt,
	})
	rep.status = http.StatusOK
	rep.body = body
	return rep
}

func unknownPath() reply {
	return reply{kind: "control", transport: "http", header: http.Header{}}.reject(errUnknownPath)
}

// methodNotAllowed mirrors chi's default: 405, empty body, an Allow header.
func methodNotAllowed(kind, transport string, allow ...string) reply {
	rep := reply{kind: kind, transport: transport, status: http.StatusMethodNotAllowed, header: http.Header{}}
	rep.header["Allow"] = allow
	return rep
}

// health answers the routes that are never captured, so a healthcheck loop
// cannot fill the store (contract section 3).
func (g *Gateway) health(r *http.Request, path string) (reply, bool) {
	rep := reply{status: http.StatusOK, header: http.Header{}}
	switch path {
	case "/", "/health", "/internal/readiness":
		rep.header.Set("Content-Type", "application/json; charset=utf-8")
		rep.body = []byte(`{"status":"ready"}`)
		if g.stopping.Load() {
			rep.status = http.StatusServiceUnavailable
			rep.body = []byte(`{"status":"stopping"}`)
		}
	case "/internal/liveness":
	case "/robots.txt":
		rep.body = []byte("User-agent: * \nDisallow: / \n")
	case "/version":
		rep.header.Set("Content-Type", "application/json; charset=utf-8")
		rep.body = []byte(`{"Version":"dev"}`)
	default:
		return reply{}, false
	}
	if r.Method != http.MethodGet {
		return methodNotAllowed("", "", http.MethodGet), true
	}
	return rep, true
}
