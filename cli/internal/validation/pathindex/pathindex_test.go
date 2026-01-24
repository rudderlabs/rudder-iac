package pathindex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPathIndexer_RealWorldYAML tests path indexing with actual RudderStack resource types
// covering simple scalars, nested objects, and sequence nodes in a single comprehensive test
func TestPathIndexer_RealWorldYAML(t *testing.T) {
	// Real-world YAML from RudderStack specs with mixed complexity
	yamlContent := `version: rudder/v0.1
kind: properties
metadata:
  name: "api_tracking"
spec:
  properties:
  - id: "api_method"
    name: "API Method"
    type: "string"
    description: "http method of the api called"
    propConfig:
      enum:
        - "GET"
        - "PUT"
        - "POST"
  - id: "http_retry_count"
    name: "HTTP Retry Count"
    type: "integer"
    description: "Number of times to retry the API call"
    propConfig:
      minimum: 0
      maximum: 10
  - id: "signin_info"
    name: "Signin Info"
    description: "Signin info"
    type: "object"`

	pi, err := NewPathIndexer([]byte(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, pi)

	// Test simple scalar values (root level)
	t.Run("simple_scalars", func(t *testing.T) {
		t.Parallel()

		pos, err := pi.PositionLookup("/version")
		require.NoError(t, err)
		assert.Equal(t, 1, pos.Line)
		assert.Equal(t, 1, pos.Column)
		assert.Equal(t, "version: rudder/v0.1", pos.LineText)

		pos, err = pi.PositionLookup("/kind")
		require.NoError(t, err)
		assert.Equal(t, 2, pos.Line)
		assert.Equal(t, 1, pos.Column)
		assert.Equal(t, "kind: properties", pos.LineText)
	})

	// Test nested object paths (metadata section)
	t.Run("nested_objects", func(t *testing.T) {
		t.Parallel()

		pos, err := pi.PositionLookup("/metadata")
		require.NoError(t, err)
		assert.Equal(t, 3, pos.Line)
		assert.Equal(t, 1, pos.Column)
		assert.Equal(t, "metadata: {...}", pos.LineText)

		pos, err = pi.PositionLookup("/metadata/name")
		require.NoError(t, err)
		assert.Equal(t, 4, pos.Line)
		assert.Equal(t, 3, pos.Column)
		assert.Equal(t, "name: api_tracking", pos.LineText)
	})

	// Test sequence/array indexing with nested objects
	t.Run("sequences_with_nested_objects", func(t *testing.T) {
		t.Parallel()

		// Array itself
		pos, err := pi.PositionLookup("/spec/properties")
		require.NoError(t, err)
		assert.Equal(t, 6, pos.Line)
		assert.Equal(t, 3, pos.Column)
		assert.Equal(t, "properties: [...]", pos.LineText)

		// First array element - simple fields
		pos, err = pi.PositionLookup("/spec/properties/0/id")
		require.NoError(t, err)
		assert.Equal(t, 7, pos.Line)
		assert.Equal(t, 5, pos.Column)
		assert.Equal(t, "id: api_method", pos.LineText)

		pos, err = pi.PositionLookup("/spec/properties/0/type")
		require.NoError(t, err)
		assert.Equal(t, 9, pos.Line)
		assert.Equal(t, 5, pos.Column)
		assert.Equal(t, "type: string", pos.LineText)

		// Nested object within first array element (propConfig)
		pos, err = pi.PositionLookup("/spec/properties/0/propConfig")
		require.NoError(t, err)
		assert.Equal(t, 11, pos.Line)
		assert.Equal(t, 5, pos.Column)
		assert.Equal(t, "propConfig: {...}", pos.LineText)

		// Nested array within object (enum)
		pos, err = pi.PositionLookup("/spec/properties/0/propConfig/enum")
		require.NoError(t, err)
		assert.Equal(t, 12, pos.Line)
		assert.Equal(t, 7, pos.Column)
		assert.Equal(t, "enum: [...]", pos.LineText)

		pos, err = pi.PositionLookup("/spec/properties/0/propConfig/enum/0")
		require.NoError(t, err)
		assert.Equal(t, 13, pos.Line)
		assert.Equal(t, 11, pos.Column)
		assert.Equal(t, "GET", pos.LineText)

		// Second array element - different nested structure
		pos, err = pi.PositionLookup("/spec/properties/1/id")
		require.NoError(t, err)
		assert.Equal(t, 16, pos.Line)
		assert.Equal(t, 5, pos.Column)
		assert.Equal(t, "id: http_retry_count", pos.LineText)

		pos, err = pi.PositionLookup("/spec/properties/1/propConfig/minimum")
		require.NoError(t, err)
		assert.Equal(t, 21, pos.Line)
		assert.Equal(t, 7, pos.Column)
		assert.Equal(t, "minimum: 0", pos.LineText)

		// Third array element - object type value
		pos, err = pi.PositionLookup("/spec/properties/2/type")
		require.NoError(t, err)
		assert.Equal(t, 26, pos.Line)
		assert.Equal(t, 5, pos.Column)
		assert.Equal(t, "type: object", pos.LineText)
	})

	// Test non-existent paths return errors
	t.Run("nonexistent_paths", func(t *testing.T) {
		pos, err := pi.PositionLookup("/nonexistent")
		assert.Error(t, err)
		assert.Nil(t, pos)
		assert.Contains(t, err.Error(), "path not found")

		pos, err = pi.PositionLookup("/spec/properties/999/id")
		assert.Error(t, err)
		assert.Nil(t, pos)
	})
}

// TestPathIndexer_EmptyYAML tests behavior with empty YAML content
func TestPathIndexer_EmptyYAML(t *testing.T) {
	pi, err := NewPathIndexer([]byte(""))
	require.NoError(t, err)
	require.NotNil(t, pi)

	// Empty YAML should result in path lookup errors
	pos, err := pi.PositionLookup("/version")
	assert.ErrorIs(t, err, ErrPathNotFound)
	assert.Nil(t, pos)
	assert.Contains(t, err.Error(), "path not found")
}

// TestPathIndexer_MalformedYAML tests error handling for invalid YAML syntax
func TestPathIndexer_MalformedYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "invalid_bracket_syntax",
			yaml: `version: [invalid`,
		},
		{
			name: "invalid_indentation",
			yaml: `version: rudder/v0.1
	kind: events`,
		},
		{
			name: "unclosed_quote",
			yaml: `name: "unclosed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi, err := NewPathIndexer([]byte(tt.yaml))
			assert.Error(t, err)
			assert.Nil(t, pi)
			assert.Contains(t, err.Error(), "parsing YAML content")
		})
	}
}

// TestPathIndexer_EdgeCases tests edge cases like empty objects, empty arrays,
// null values, and various empty node types
func TestPathIndexer_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		yaml            string
		expectErr       bool
		positionLookups []positionCheck
	}{
		{
			name: "empty_objects_and_arrays",
			yaml: `spec:
  metadata: {}
  config: {}
  events: []
  properties: []`,
			expectErr: false,
			positionLookups: []positionCheck{
				{path: "/spec", expectedLine: 1, expectedLineText: "spec: {...}"},
				{path: "/spec/metadata", expectedLine: 2, expectedLineText: "metadata: {}"},
				{path: "/spec/config", expectedLine: 3, expectedLineText: "config: {}"},
				{path: "/spec/events", expectedLine: 4, expectedLineText: "events: [...]"},
				{path: "/spec/properties", expectedLine: 5, expectedLineText: "properties: [...]"},
			},
		},
		{
			name: "null_and_empty_string_values",
			yaml: `version: ""
kind:
metadata:
  name: ""
  group:`,
			expectErr: false,
			positionLookups: []positionCheck{
				{path: "/version", expectedLine: 1, expectedLineText: "version: "},
				{path: "/kind", expectedLine: 2, expectedLineText: "kind: "},
				{path: "/metadata/name", expectedLine: 4, expectedLineText: "name: "},
				{path: "/metadata/group", expectedLine: 5, expectedLineText: "group: "},
			},
		},
		{
			name: "mixed_empty_and_populated_arrays",
			yaml: `spec:
  empty_list: []
  populated_list:
    - id: "item1"
    - id: "item2"`,
			expectErr: false,
			positionLookups: []positionCheck{
				{path: "/spec/empty_list", expectedLine: 2, expectedLineText: "empty_list: [...]"},
				{path: "/spec/populated_list", expectedLine: 3, expectedLineText: "populated_list: [...]"},
				{path: "/spec/populated_list/0/id", expectedLine: 4, expectedLineText: "id: item1"},
				{path: "/spec/populated_list/1/id", expectedLine: 5, expectedLineText: "id: item2"},
			},
		},
		{
			name: "nested_empty_structures",
			yaml: `parent:
  child1:
    grandchild: {}
  child2:
    items: []`,
			expectErr: false,
			positionLookups: []positionCheck{
				{path: "/parent", expectedLine: 1, expectedLineText: "parent: {...}"},
				{path: "/parent/child1", expectedLine: 2, expectedLineText: "child1: {...}"},
				{path: "/parent/child1/grandchild", expectedLine: 3, expectedLineText: "grandchild: {}"},
				{path: "/parent/child2/items", expectedLine: 5, expectedLineText: "items: [...]"},
			},
		},
		{
			name: "comments_with_empty_values",
			yaml: `# Top level comment
version: ""  # inline comment
# Comment before empty object
metadata: {}
# Comment before empty array
events: []`,
			expectErr: false,
			positionLookups: []positionCheck{
				{path: "/version", expectedLine: 2, expectedLineText: "version: "},
				{path: "/metadata", expectedLine: 4, expectedLineText: "metadata: {}"},
				{path: "/events", expectedLine: 6, expectedLineText: "events: [...]"},
			},
		},
		{
			name: "scalar_types_with_various_values",
			yaml: `string_val: "test"
number_val: 42
float_val: 3.14
bool_true: true
bool_false: false
null_val: null`,
			expectErr: false,
			positionLookups: []positionCheck{
				{path: "/string_val", expectedLine: 1, expectedLineText: "string_val: test"},
				{path: "/number_val", expectedLine: 2, expectedLineText: "number_val: 42"},
				{path: "/float_val", expectedLine: 3, expectedLineText: "float_val: 3.14"},
				{path: "/bool_true", expectedLine: 4, expectedLineText: "bool_true: true"},
				{path: "/bool_false", expectedLine: 5, expectedLineText: "bool_false: false"},
				{path: "/null_val", expectedLine: 6, expectedLineText: "null_val: null"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi, err := NewPathIndexer([]byte(tt.yaml))

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, pi)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pi)

			for _, check := range tt.positionLookups {
				pos, err := pi.PositionLookup(check.path)
				require.NoError(t, err, "Path %s should exist", check.path)
				require.NotNil(t, pos, "Position should not be nil for path %s", check.path)
				assert.Equal(t, check.expectedLine, pos.Line, "Line mismatch for path %s", check.path)
				assert.Equal(t, check.expectedLineText, pos.LineText, "LineText mismatch for path %s", check.path)
			}
		})
	}
}

type positionCheck struct {
	path             string
	expectedLine     int
	expectedLineText string
}

// TestPathIndexer_NearestPosition tests the fallback behavior when exact paths don't exist
func TestPathIndexer_NearestPosition(t *testing.T) {
	// Same YAML fixture used in TestPathIndexer_RealWorldYAML for consistency
	yamlContent := `version: rudder/v0.1
kind: properties
metadata:
  name: "api_tracking"
spec:
  properties:
  - id: "api_method"
    name: "API Method"
    type: "string"
    description: "http method of the api called"
    propConfig:
      enum:
        - "GET"
        - "PUT"
        - "POST"
  - id: "http_retry_count"
    name: "HTTP Retry Count"
    type: "integer"
    description: "Number of times to retry the API call"
    propConfig:
      minimum: 0
      maximum: 10
  - id: "signin_info"
    name: "Signin Info"
    description: "Signin info"
    type: "object"`

	pi, err := NewPathIndexer([]byte(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, pi)

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()

		// When path exists, NearestPosition returns exact position
		pos := pi.NearestPosition("/spec/properties")
		require.NotNil(t, pos)
		assert.Equal(t, 6, pos.Line)
		assert.Equal(t, "properties: [...]", pos.LineText)
	})

	t.Run("ancestor resolution", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name             string
			path             string
			expectedLine     int
			expectedLineText string
		}{
			{
				name:             "one_level_up",
				path:             "/spec/nonexistent",
				expectedLine:     5,
				expectedLineText: "spec: {...}",
			},
			{
				name:             "multiple_levels_up",
				path:             "/spec/properties/0/nonexistent/deep/path",
				expectedLine:     7,
				expectedLineText: "- id: api_method",
			},
			{
				name:             "invalid_array_index",
				path:             "/spec/properties/999/id",
				expectedLine:     6,
				expectedLineText: "properties: [...]",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				pos := pi.NearestPosition(tt.path)
				require.NotNil(t, pos, "NearestPosition should never return nil")
				assert.Equal(t, tt.expectedLine, pos.Line)
				assert.Equal(t, tt.expectedLineText, pos.LineText)
			})
		}
	})

	t.Run("root fallback", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			path string
		}{
			{name: "root_path_itself", path: "/"},
			{name: "complete_miss", path: "/nonexistent/path/deep"},
			{name: "single_segment_miss", path: "/nonexistent"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				pos := pi.NearestPosition(tt.path)
				require.NotNil(t, pos, "NearestPosition should never return nil")
				assert.Equal(t, 1, pos.Line)
				assert.Equal(t, 1, pos.Column)
				assert.Equal(t, "", pos.LineText)
			})
		}
	})
}
