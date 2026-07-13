package client_test

import (
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError *client.APIError
		want     string
	}{
		{
			name: "uses Message field",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Invalid request",
				ErrorCode:      "BAD_REQUEST",
			},
			want: "http status code: 400, error code: 'BAD_REQUEST', error: 'Invalid request'",
		},
		{
			name: "uses ErrorMessage when Message is empty",
			apiError: &client.APIError{
				HTTPStatusCode: 404,
				ErrorMessage:   "Resource not found",
				ErrorCode:      "NOT_FOUND",
			},
			want: "http status code: 404, error code: 'NOT_FOUND', error: 'Resource not found'",
		},
		{
			name: "prefers Message over ErrorMessage",
			apiError: &client.APIError{
				HTTPStatusCode: 500,
				Message:        "Internal server error",
				ErrorMessage:   "Server error",
				ErrorCode:      "INTERNAL_ERROR",
			},
			want: "http status code: 500, error code: 'INTERNAL_ERROR', error: 'Internal server error'",
		},
		{
			name: "details with single field",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "request validation failed",
				Details:        json.RawMessage(`{"mappings":"Array must contain at least 1 element(s)"}`),
			},
			want: "http status code: 400, error code: '', error: 'request validation failed' (mappings: Array must contain at least 1 element(s))",
		},
		{
			name: "details with multiple fields renders sorted by key",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "request validation failed",
				Details:        json.RawMessage(`{"name":"Required","mappings":"Array must contain at least 1 element(s)"}`),
			},
			want: "http status code: 400, error code: '', error: 'request validation failed' (mappings: Array must contain at least 1 element(s), name: Required)",
		},
		{
			name: "empty details object is omitted",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Invalid request",
				ErrorCode:      "BAD_REQUEST",
				Details:        json.RawMessage(`{}`),
			},
			want: "http status code: 400, error code: 'BAD_REQUEST', error: 'Invalid request'",
		},
		{
			name: "null details is omitted",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Invalid request",
				ErrorCode:      "BAD_REQUEST",
				Details:        json.RawMessage(`null`),
			},
			want: "http status code: 400, error code: 'BAD_REQUEST', error: 'Invalid request'",
		},
		{
			name: "nil details is omitted",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Invalid request",
				ErrorCode:      "BAD_REQUEST",
			},
			want: "http status code: 400, error code: 'BAD_REQUEST', error: 'Invalid request'",
		},
		{
			name: "non-object details falls back to raw JSON",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "request validation failed",
				Details:        json.RawMessage(`["a","b"]`),
			},
			want: `http status code: 400, error code: '', error: 'request validation failed' (details: ["a","b"])`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.apiError.Error())
		})
	}
}

func TestAPIError_IsFeatureNotEnabled(t *testing.T) {
	tests := []struct {
		name     string
		apiError *client.APIError
		want     bool
	}{
		{
			name: "returns true for 403 with feature flag message",
			apiError: &client.APIError{
				HTTPStatusCode: 403,
				Message:        "Flag is not enabled for your account",
			},
			want: true,
		},
		{
			name: "checks ErrorMessage when Message is empty",
			apiError: &client.APIError{
				HTTPStatusCode: 403,
				ErrorMessage:   "Flag is not enabled for your account",
			},
			want: true,
		},
		{
			name: "returns true for 403 with feature message",
			apiError: &client.APIError{
				HTTPStatusCode: 403,
				Message:        "Feature is not enabled for your account: DATA_GRAPH",
			},
			want: true,
		},
		{
			name: "checks ErrorMessage for feature message",
			apiError: &client.APIError{
				HTTPStatusCode: 403,
				ErrorMessage:   "Feature is not enabled for your account: DATA_GRAPH",
			},
			want: true,
		},
		{
			name: "returns false for 403 with different message",
			apiError: &client.APIError{
				HTTPStatusCode: 403,
				Message:        "Insufficient permissions",
			},
			want: false,
		},
		{
			name: "returns false when flag-disabled status is not 403",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Flag is not enabled for your account",
			},
			want: false,
		},
		{
			name: "returns false when feature-disabled status is not 403",
			apiError: &client.APIError{
				HTTPStatusCode: 500,
				Message:        "Feature is not enabled for your account: DATA_GRAPH",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.apiError.IsFeatureNotEnabled())
		})
	}
}
