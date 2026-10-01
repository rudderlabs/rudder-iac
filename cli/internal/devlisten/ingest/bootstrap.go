package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"slices"
)

const jsonContentType = "application/json; charset=utf-8"

func jsonReply(status int, body string) reply {
	return reply{status: status, contentType: jsonContentType, body: []byte(body)}
}

// proxyReply answers the proxy routes. RudderStack sends no CORS headers on
// them.
func proxyReply(path string) (reply, bool) {
	switch path {
	case "/cluster-info":
		return jsonReply(http.StatusOK, `{"nodeCount":1}`), true
	case "/v1/webhook":
		return reject(errProxyDisabled), true
	}
	return reply{}, false
}

// healthReply answers the routes a supervisor polls. They are not captured,
// so a health check cannot fill the store.
func (h *Handler) healthReply(r *http.Request) (reply, bool) {
	var rep reply
	switch r.URL.Path {
	case "/", "/health", "/internal/readiness":
		rep = jsonReply(http.StatusOK, `{"status":"ready"}`)
	case "/internal/liveness":
		rep = reply{status: http.StatusOK}
	case "/robots.txt":
		rep = reply{status: http.StatusOK, body: []byte("User-agent: * \nDisallow: / \n")}
	case "/version":
		body, _ := json.Marshal(struct{ Version string }{h.version})
		rep = jsonReply(http.StatusOK, string(body))
	default:
		return reply{}, false
	}
	if r.Method != http.MethodGet {
		return methodNotAllowed(http.MethodGet), true
	}
	return rep, true
}

// control answers the routes an SDK calls besides ingestion.
func (h *Handler) control(r *http.Request, c *Capture) reply {
	switch r.URL.Path {
	case "/sourceConfig", "/sourceConfig/":
		return h.sourceConfig(r, c.WriteKey)
	case "/rsaMetrics":
		return jsonReply(http.StatusOK, `{}`)
	}
	return reject(errUnknownPath)
}

// sourceConfig answers as the control plane does for an enabled source with
// no destinations, so browser and mobile SDKs start and send to the listener.
func (h *Handler) sourceConfig(r *http.Request, writeKey string) reply {
	switch {
	case r.Method == http.MethodHead && queryValue(r.URL.RawQuery, "view") == "ad":
		return reply{status: http.StatusNoContent}
	case r.Method != http.MethodGet && r.Method != http.MethodHead:
		return methodNotAllowed("GET, HEAD")
	case len(h.writeKeys) > 0 && writeKey == "":
		return configRejection(http.StatusUnauthorized, "Writekey not found in basic auth")
	case len(h.writeKeys) > 0 && !slices.Contains(h.writeKeys, writeKey):
		return configRejection(http.StatusBadRequest, "Invalid write key")
	}

	updatedAt := h.startedAt.UTC().Format("2006-01-02T15:04:05.000Z")
	sum := sha256.Sum256([]byte(writeKey))
	disabled := map[string]any{"enabled": false}
	// A map marshals with sorted keys, so the bytes do not depend on field order.
	body, _ := json.Marshal(map[string]any{
		"source": map[string]any{
			"config":       map[string]any{"statsCollection": map[string]any{"errors": disabled, "metrics": disabled}},
			"dataplanes":   map[string]any{},
			"destinations": []any{},
			"enabled":      true,
			"id":           "dev-" + hex.EncodeToString(sum[:6]),
			"name":         "rudder-cli dev listen",
			"updatedAt":    updatedAt,
			"workspaceId":  "dev-workspace",
			"writeKey":     writeKey,
		},
		"updatedAt": updatedAt,
	})
	rep := jsonReply(http.StatusOK, string(body))
	if r.Method == http.MethodHead {
		rep.body = nil
	}
	return rep
}

func configRejection(status int, message string) reply {
	rep := jsonReply(status, `{"message":"`+message+`"}`)
	rep.rejection = &Rejection{Stage: "auth", Reason: message}
	return rep
}
