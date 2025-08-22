package helpers

import (
	"fmt"
	"reflect"
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

func TestSortSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      any
		expected   []string // String representation of sorted elements
		shouldSort bool     // Whether the slice should be sorted or maintain original order
	}{
		{
			name:       "string slice should be sorted alphabetically",
			input:      []string{"zebra", "apple", "banana"},
			expected:   []string{"apple", "banana", "zebra"},
			shouldSort: true,
		},
		{
			name:       "integer slice should be sorted numerically by string representation",
			input:      []int{30, 1, 20},
			expected:   []string{"1", "20", "30"},
			shouldSort: true,
		},
		{
			name:       "mixed primitive type slice should be sorted by string representation",
			input:      []any{30, "apple", 1, "zebra"},
			expected:   []string{"1", "30", "apple", "zebra"},
			shouldSort: true,
		},
		{
			name:       "slice with maps without name key should maintain original order",
			input:      []map[string]any{{"id": "2"}, {"id": "1"}},
			expected:   []string{"map[id:2]", "map[id:1]"}, // Original order preserved
			shouldSort: false,
		},
		{
			name:       "map slice with name key should be sorted by name",
			input:      []map[string]any{{"name": "zebra", "id": "3"}, {"name": "apple", "id": "1"}, {"name": "banana", "id": "2"}},
			expected:   []string{"map[id:1 name:apple]", "map[id:2 name:banana]", "map[id:3 name:zebra]"},
			shouldSort: true,
		},
		{
			name:       "interface slice with maps having name key should be sorted",
			input:      []any{map[string]any{"name": "zebra"}, map[string]any{"name": "apple"}},
			expected:   []string{"map[name:apple]", "map[name:zebra]"},
			shouldSort: true,
		},
		{
			name:       "map slice with partial name keys groups maps with name key first, sorted",
			input:      []map[string]any{{"name": "bazebra"}, {"id": "1"}, {"name": "abapple"}},
			expected:   []string{"map[name:abapple]", "map[name:bazebra]", "map[id:1]"},
			shouldSort: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sorted := sortSlice(reflect.ValueOf(tt.input))

			result := make([]string, sorted.Len())
			for i := 0; i < sorted.Len(); i++ {
				result[i] = fmt.Sprintf("%v", sorted.Index(i).Interface())
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSliceOrderable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{
			name:     "string slice is orderable",
			input:    []string{"a", "b"},
			expected: true,
		},
		{
			name:     "int slice is orderable",
			input:    []int{1, 2},
			expected: true,
		},
		{
			name:     "float slice is orderable",
			input:    []float64{1.1, 2.2},
			expected: true,
		},
		{
			name:     "bool slice is orderable",
			input:    []bool{true, false},
			expected: true,
		},
		{
			name:     "interface slice with primitives is orderable",
			input:    []any{"hello", 42},
			expected: true,
		},
		{
			name:     "map slice is orderable",
			input:    []map[string]any{{"key": "value"}},
			expected: true,
		},
		{
			name:     "struct slice is not orderable",
			input:    []struct{ Name string }{{"test"}},
			expected: false,
		},
		{
			name:     "slice of slices is not orderable",
			input:    [][]string{{"a"}, {"b"}},
			expected: false,
		},
		{
			name:     "empty slice is orderable",
			input:    []string{},
			expected: false,
		},
		{
			name:     "interface slice with map is orderable",
			input:    []any{map[string]any{"key": "value"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, _ := isSliceOrderable(reflect.ValueOf(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}
