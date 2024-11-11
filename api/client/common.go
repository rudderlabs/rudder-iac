package client

import (
	"encoding/json"
	"fmt"
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
