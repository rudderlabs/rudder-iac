package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	FeatureFlagNotEnabled = "Flag is not enabled for your account"
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
	msg := e.Message
	if msg == "" {
		msg = e.ErrorMessage
	}
	return fmt.Sprintf("http status code: %d, error code: '%s', error: '%s'", e.HTTPStatusCode, e.ErrorCode, msg)
}

func (e *APIError) FeatureNotEnabled() bool {
	msg := e.Message
	if msg == "" {
		msg = e.ErrorMessage
	}
	return e.HTTPStatusCode == 403 && strings.Contains(msg, FeatureFlagNotEnabled)
}
