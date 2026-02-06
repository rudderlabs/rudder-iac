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
	return fmt.Sprintf("http status code: %d, error code: '%s', error: '%s'", e.HTTPStatusCode, e.ErrorCode, e.Msg())
}

func (e *APIError) FeatureNotEnabled() bool {
	return e.HTTPStatusCode == 403 && strings.Contains(e.Msg(), FeatureFlagNotEnabled)
}

func (e *APIError) Msg() string {
	if e.Message != "" {
		return e.Message
	}
	return e.ErrorMessage
}
