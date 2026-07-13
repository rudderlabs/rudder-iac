package client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	FeatureFlagNotEnabledMessagePrefix = "Flag is not enabled for your account"
	FeatureNotEnabledMessagePrefix     = "Feature is not enabled for your account"
)

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
	msg := fmt.Sprintf("http status code: %d, error code: '%s', error: '%s'", e.HTTPStatusCode, e.ErrorCode, e.Msg())
	if suffix := formatDetails(e.Details); suffix != "" {
		msg += " " + suffix
	}
	return msg
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

func (e *APIError) IsFeatureNotEnabled() bool {
	return e.HTTPStatusCode == 403 &&
		(strings.Contains(e.Msg(), FeatureFlagNotEnabledMessagePrefix) ||
			strings.Contains(e.Msg(), FeatureNotEnabledMessagePrefix))
}

func (e *APIError) Msg() string {
	if e.Message != "" {
		return e.Message
	}
	return e.ErrorMessage
}
