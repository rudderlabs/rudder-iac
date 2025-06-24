package helpers

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareStates(t *testing.T) {
	expected := map[string]any{
		"id":   "public_api_request",
		"type": "event",
		"input": map[string]any{
			"categoryId":  nil,
			"description": "This event is triggered everytime a public API is requested",
			"eventType":   "track",
			"name":        "Public API Requested",
		},
		"output": map[string]any{
			"categoryId":  nil,
			"createdAt":   "2025-02-19 07:10:24.15 +0000 UTC",
			"description": "This event is triggered everytime a public API is requested",
			"eventType":   "track",
			"id":          "ev_2tFXvEpGOcjHChUcwyyyyyyy",
			"name":        "Public API Requested",
			"updatedAt":   "2025-02-19 07:10:24.15 +0000 UTC",
			"workspaceId": "2fmqH5sdM4QJocMlxxxxxxxx",
		},
		"dependencies": nil,
	}

	tests := []struct {
		name      string
		actual    any
		expected  any
		ignore    []string
		expectErr bool
		errMsg    []string
	}{
		{
			name:     "identical states result in no comparison errors",
			actual:   expected,
			expected: expected,
			ignore:   []string{},
		},
		{
			name:     "state comparison with different data but added in ignore list result in no errors",
			expected: expected,
			actual: map[string]any{
				"id":   "public_api_request",
				"type": "event",
				"input": map[string]any{
					"categoryId":  nil,
					"description": "This event is triggered everytime a public API is requested",
					"eventType":   "track",
					"name":        "Public API Requested",
				},
				"output": map[string]any{
					"categoryId":  nil,
					"createdAt":   "2024-01-18 07:10:24.15 +0000 UTC",
					"description": "This event is triggered everytime a public API is requested",
					"eventType":   "track",
					"id":          "ev_2tFXvEpGOcjHChUcwVHfLCaxxxx",
					"name":        "Public API Requested",
					"updatedAt":   "2024-01-18 08:10:24.15 +0000 UTC",
					"workspaceId": "2fmqH5sdM4QJocMl3fyyyyyy", // this difference is ignored
				},
				"dependencies": nil,
			},
			ignore:    []string{"output.id", "output.createdAt", "output.updatedAt", "output.workspaceId"},
			expectErr: false,
		},
		{
			name:     "state comparison with differing data errors",
			expected: expected,
			actual: map[string]any{
				"id":   "public_api_request",
				"type": "event",
				"input": map[string]any{
					"categoryId":  "cat_2tFXvEpGOcjHChUcwVHfLCa5lI6", // categoryId mismatch error
					"description": "This event is triggered everytime a public API is requested",
					"eventType":   "track",
					"name":        "Public API Requested New", // name mismatch error
				},
				"output": map[string]any{
					"categoryId":  nil,
					"createdAt":   "2024-01-18 07:10:24.15 +0000 UTC",
					"description": "This event is triggered everytime a public API is requested",
					"eventType":   "track",
					"id":          "ev_2tFXvEpGOcjHChUcwVHfLCaxxxx",
					"name":        "Public API Requested",
					"updatedAt":   "2024-01-18 08:10:24.15 +0000 UTC",
					"workspaceId": "2fmqH5sdM4QJocMl3fyyyyyy", // this difference is ignored
				},
				"dependencies": nil,
			},
			ignore:    []string{"output.id", "output.createdAt", "output.updatedAt", "output.workspaceId"},
			expectErr: true,
			errMsg: []string{
				"mismatch at path 'input.name': got Public API Requested New, want Public API Requested",
				"mismatch at path 'input.categoryId': got cat_2tFXvEpGOcjHChUcwVHfLCa5lI6, want <nil>",
			},
		},
		{
			name:     "state comparison with additional keys in actual result in comparison errors",
			expected: expected,
			actual: map[string]any{
				"id":   "public_api_request",
				"type": "event",
				"input": map[string]any{
					"categoryId":  nil,
					"description": "This event is triggered everytime a public API is requested",
					"eventType":   "track",
					"name":        "Public API Requested",
				},
				"output": map[string]any{
					"categoryId":  nil,
					"createdAt":   "2024-01-18 07:10:24.15 +0000 UTC",
					"description": "This event is triggered everytime a public API is requested",
					"eventType":   "track",
					"id":          "ev_2tFXvEpGOcjHChUcwVHfLCaxxxx",
					"name":        "Public API Requested",
					"updatedAt":   "2024-01-18 08:10:24.15 +0000 UTC",
					"workspaceId": "2fmqH5sdM4QJocMl3fyyyyyy", // this difference is ignored
				},
				"dependencies": nil,
				"extraKey":     "extraValue",
			},
			ignore:    []string{"output.id", "output.createdAt", "output.updatedAt", "output.workspaceId"},
			expectErr: true,
			errMsg: []string{
				"mismatch at path '': map key count differs, got 6 keys, want 5 keys",
				"mismatch at path '': extra key 'extraKey' in actual",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CompareStates(tt.actual, tt.expected, tt.ignore)
			if !tt.expectErr {
				require.Nil(t, err)
				return
			}

			require.NotNil(t, err)

			actualErrMsgs := strings.Split(err.Error(), "\n")
			assert.Equal(t, len(tt.errMsg), len(actualErrMsgs))
			for _, errMsg := range tt.errMsg {
				assert.True(t, slices.Contains(actualErrMsgs, errMsg), "expected error message %s not found in %v", errMsg, actualErrMsgs)
			}
		})
	}
}
