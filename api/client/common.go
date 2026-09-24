package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const (
	FeatureFlagNotEnabledMessagePrefix = "Flag is not enabled for your account"
	FeatureNotEnabledMessagePrefix     = "Feature is not enabled for your account"
	// Raised on create when the resource's definition is gated behind flags the
	// account lacks, e.g. `destination "SNOWPIPE_STREAMING" is not available for your account`.
	ResourceNotAvailableMessageSuffix = "is not available for your account"
)

// blockedByConnectionsMessages are the refusals the control plane raises when a
// source or destination still has connections. Matching on prose is fragile,
// but it is the only signal available: BadRequestError and ValidationError
// carry a message and a status and nothing else, so APIError.ErrorCode is empty
// on all three paths. Each entry is a distinct service with its own wording:
//
//	destination.service.ts    deleting a destination
//	source.service.ts         deleting an event-stream source
//	retl/service.ts           deleting an rETL source
//
// Delete refusals only. retl/service.ts raises "connected to some destinations"
// on an update too ("... Cannot update modelPath"), which this would match — no
// update path calls it today, and one should not without widening this comment.
var blockedByConnectionsMessages = []string{
	"active connections",
	"connected to some destinations",
}

type Paging struct {
	Total int    `json:"total"`
	Next  string `json:"next"`
}

type APIPage struct {
	Paging Paging `json:"paging"`
}

type APIError struct {
	HTTPStatusCode int
	Message        string          `json:"error"`
	ErrorMessage   string          `json:"message"` // Some APIs use "message" instead of "error"
	ErrorCode      string          `json:"code"`
	Details        json.RawMessage `json:"details"`
}

func (e *APIError) Error() string {
	reason := fmt.Sprintf("'%s'", e.Msg())
	if suffix := formatDetails(e.Details); suffix != "" {
		reason += " " + suffix
	}

	// Both 403 causes need the user to act, with a token that has the permission
	// or an account that has the feature, so name that rather than the status line.
	if e.FeatureFlagNotEnabled() {
		return fmt.Sprintf("feature not enabled: %s: contact RudderStack support to enable it", reason)
	}
	if e.HTTPStatusCode == http.StatusForbidden {
		return fmt.Sprintf("permission denied: %s: use an access token with the required permissions, or ask a workspace admin to grant them", reason)
	}

	return fmt.Sprintf("http status code: %d, error code: '%s', error: %s", e.HTTPStatusCode, e.ErrorCode, reason)
}

// formatDetails renders APIError.Details as a parenthesised suffix.
// The common shape from rudder-api is a flat {"field":"message"} object; keys
// are sorted so the output is deterministic. Anything else (arrays, nested
// objects, primitives, malformed JSON) falls back to the raw payload so we
// never lose information.
func formatDetails(details json.RawMessage) string {
	trimmed := strings.TrimSpace(string(details))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" || trimmed == "[]" {
		return ""
	}

	var fields map[string]string
	if err := json.Unmarshal(details, &fields); err == nil && len(fields) > 0 {
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, fields[k]))
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}

	return "(details: " + trimmed + ")"
}

func (e *APIError) FeatureFlagNotEnabled() bool {
	return e.HTTPStatusCode == 403 &&
		(strings.Contains(e.Msg(), FeatureFlagNotEnabledMessagePrefix) ||
			strings.Contains(e.Msg(), FeatureNotEnabledMessagePrefix) ||
			strings.Contains(e.Msg(), ResourceNotAvailableMessageSuffix))
}

// BlockedByConnections reports whether this is the control plane refusing to
// delete a source or destination because connections still point at it.
func (e *APIError) BlockedByConnections() bool {
	if e.HTTPStatusCode != http.StatusBadRequest {
		return false
	}

	msg := strings.ToLower(e.Msg())
	for _, want := range blockedByConnectionsMessages {
		if strings.Contains(msg, want) {
			return true
		}
	}

	return false
}

func (e *APIError) Msg() string {
	if e.Message != "" {
		return e.Message
	}
	return e.ErrorMessage
}
