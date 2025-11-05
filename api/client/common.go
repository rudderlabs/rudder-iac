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
	ErrorCode      string          `json:"code"`
	Details        json.RawMessage `json:"details"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("http status code: %d, error code: '%s', error: '%s'", e.HTTPStatusCode, e.ErrorCode, e.Message)
}

func (e *APIError) FeatureNotEnabled() bool {
	return e.HTTPStatusCode == 403 && strings.Contains(e.Message, FeatureFlagNotEnabled)
}
