package state

import (
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name     string
		state    *State
		expected string
	}{
		{
			name: "empty state",
			state: &State{
				Resources: map[string]*ResourceState{},
			},
			expected: `{"resources":{}, "version": "1.0.0"}`,
		},
		{
			name: "basic resource",
			state: &State{
				Resources: map[string]*ResourceState{
					"test:resource": {
						ID:   "123",
						Type: "test",
						Input: map[string]interface{}{
							"name": "test-resource",
						},
						Output: map[string]interface{}{
							"id": "123",
						},
						Dependencies: []string{"test:dep1"},
					},
				},
			},
			expected: `{"resources":{"test:resource":{"id":"123","type":"test","input":{"name":"test-resource"},"output":{"id":"123"},"dependencies":["test:dep1"]}}, "version": "1.0.0"}`,
		},
		{
			name: "with property references",
			state: &State{
				Resources: map[string]*ResourceState{
					"test:resource": {
						ID:   "123",
						Type: "test",
						Input: map[string]interface{}{
							"ref": resources.PropertyRef{
								URN:      "test:dep1",
								Property: "id",
							},
						},
						Output: map[string]interface{}{},
					},
				},
			},
			expected: `{"resources":{"test:resource":{"id":"123","type":"test","input":{"ref":{"$ref":"test:dep1","property":"id"}},"output":{},"dependencies":null}}, "version": "1.0.0"}`,
		},
		{
			name: "with nested structures",
			state: &State{
				Resources: map[string]*ResourceState{
					"test:resource": {
						ID:   "123",
						Type: "test",
						Input: map[string]interface{}{
							"nested": map[string]interface{}{
								"ref": resources.PropertyRef{
									URN:      "test:dep1",
									Property: "id",
								},
							},
							"array": []interface{}{
								map[string]interface{}{
									"ref": resources.PropertyRef{
										URN:      "test:dep2",
										Property: "name",
									},
								},
								"simple-value",
							},
						},
						Output: map[string]interface{}{},
					},
				},
			},
			expected: `{"resources":{"test:resource":{"id":"123","type":"test","input":{"array":[{"ref":{"$ref":"test:dep2","property":"name"}},"simple-value"],"nested":{"ref":{"$ref":"test:dep1","property":"id"}}},"output":{},"dependencies":null}}, "version": "1.0.0"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToJSON(tt.state)
			require.NoError(t, err)

			// Compare JSON strings after normalizing
			var expectedJSON, resultJSON interface{}
			err = json.Unmarshal([]byte(tt.expected), &expectedJSON)
			require.NoError(t, err)
			err = json.Unmarshal(result, &resultJSON)
			require.NoError(t, err)

			assert.Equal(t, expectedJSON, resultJSON)
		})
	}
}

func TestEncodeDecodeResourceState(t *testing.T) {
	tests := []struct {
		name            string
		resource        *ResourceState
		expectedEncoded *ResourceState
	}{
		{
			name: "basic resource",
			resource: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"name": "test-resource",
				},
				Output: map[string]interface{}{
					"id": "123",
				},
				Dependencies: []string{"test:dep1"},
			},
			expectedEncoded: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"name": "test-resource",
				},
				Output: map[string]interface{}{
					"id": "123",
				},
				Dependencies: []string{"test:dep1"},
			},
		},
		{
			name: "with property references",
			resource: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"ref": resources.PropertyRef{
						URN:      "test:dep1",
						Property: "id",
					},
				},
				Output: map[string]interface{}{},
			},
			expectedEncoded: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"ref": map[string]interface{}{
						"$ref":     "test:dep1",
						"property": "id",
					},
				},
				Output: map[string]interface{}{},
			},
		},
		{
			name: "with nested structures",
			resource: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"nested": map[string]interface{}{
						"ref": resources.PropertyRef{
							URN:      "test:dep1",
							Property: "id",
						},
					},
					"array": []interface{}{
						map[string]interface{}{
							"ref": resources.PropertyRef{
								URN:      "test:dep2",
								Property: "name",
							},
						},
						"simple-value",
					},
				},
				Output: map[string]interface{}{},
			},
			expectedEncoded: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"nested": map[string]interface{}{
						"ref": map[string]interface{}{
							"$ref":     "test:dep1",
							"property": "id",
						},
					},
					"array": []interface{}{
						map[string]interface{}{
							"ref": map[string]interface{}{
								"$ref":     "test:dep2",
								"property": "name",
							},
						},
						"simple-value",
					},
				},
				Output: map[string]interface{}{},
			},
		},
		{
			name: "with advanced nested structures",
			resource: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"nested": []map[string]interface{}{
						{
							"ref": resources.PropertyRef{
								URN:      "test:dep1",
								Property: "id",
							},
						},
						{
							"ref": resources.PropertyRef{
								URN:      "test:dep2",
								Property: "id",
							},
						},
					},
					"array": []interface{}{
						resources.PropertyRef{
							URN:      "test:dep3",
							Property: "name",
						},
						"simple-value",
					},
				},
				Output: map[string]interface{}{},
			},
			expectedEncoded: &ResourceState{
				ID:   "123",
				Type: "test",
				Input: map[string]interface{}{
					"nested": []map[string]interface{}{
						{
							"ref": map[string]interface{}{
								"$ref":     "test:dep1",
								"property": "id",
							},
						},
						{
							"ref": map[string]interface{}{
								"$ref":     "test:dep2",
								"property": "id",
							},
						},
					},
					"array": []interface{}{
						map[string]interface{}{
							"$ref":     "test:dep3",
							"property": "name",
						},
						"simple-value",
					},
				},
				Output: map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encoding
			encoded := EncodeResourceState(tt.resource)
			assert.Equal(t, tt.expectedEncoded, encoded)

			// Test decoding
			decoded := DecodeResourceState(encoded)
			assert.Equal(t, tt.resource, decoded)
		})
	}
}
