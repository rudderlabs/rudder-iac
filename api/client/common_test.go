package client_test

import (
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.apiError.Error())
		})
	}
}

func TestAPIError_FeatureNotEnabled(t *testing.T) {
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
			name: "returns false for 403 with different message",
			apiError: &client.APIError{
				HTTPStatusCode: 403,
				Message:        "Insufficient permissions",
			},
			want: false,
		},
		{
			name: "returns false when status is not 403",
			apiError: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Flag is not enabled for your account",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.apiError.FeatureNotEnabled())
		})
	}
}
