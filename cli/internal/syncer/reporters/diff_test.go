package reporters

import (
	"reflect"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestComputeNestedDiffs_Maps(t *testing.T) {
	t.Parallel()

	mockProperyRefFunc := func(state any) (string, error) {
		return "", nil
	}

	mockPropertRef1 := &resources.PropertyRef{URN: "some:urn1", Resolve: mockProperyRefFunc}
	mockPropertRef2 := &resources.PropertyRef{URN: "some:urn2", Resolve: mockProperyRefFunc}

	tests := []struct {
		name     string
		source   any
		target   any
		expected map[string]ValuePair
	}{
		{
			name:   "simple map with one changed field",
			source: map[string]any{"a": 1, "b": 2},
			target: map[string]any{"a": 1, "b": 3},
			expected: map[string]ValuePair{
				"b": {Source: 2, Target: 3},
			},
		},
		{
			name:   "multiple changed fields",
			source: map[string]any{"a": 1, "b": 2, "c": 3},
			target: map[string]any{"a": 10, "b": 2, "c": 30},
			expected: map[string]ValuePair{
				"a": {Source: 1, Target: 10},
				"c": {Source: 3, Target: 30},
			},
		},
		{
			name:   "deep nesting (3 levels)",
			source: map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
			target: map[string]any{"a": map[string]any{"b": map[string]any{"c": 2}}},
			expected: map[string]ValuePair{
				"a.b.c": {Source: 1, Target: 2},
			},
		},
		{
			name:   "added keys (nil → value)",
			source: map[string]any{"a": 1},
			target: map[string]any{"a": 1, "b": 2},
			expected: map[string]ValuePair{
				"b": {Source: nil, Target: 2},
			},
		},
		{
			name:   "removed keys (value → nil)",
			source: map[string]any{"a": 1, "b": 2},
			target: map[string]any{"a": 1},
			expected: map[string]ValuePair{
				"b": {Source: 2, Target: nil},
			},
		},
		{
			name:     "no changes (empty result)",
			source:   map[string]any{"a": 1, "b": 2},
			target:   map[string]any{"a": 1, "b": 2},
			expected: map[string]ValuePair{},
		},
		{
			name:   "property refs (pointers)",
			source: map[string]any{"a": mockPropertRef1, "b": 2},
			target: map[string]any{"a": mockPropertRef2, "b": 2},
			expected: map[string]ValuePair{
				"a": {
					Source: mockPropertRef1,
					Target: mockPropertRef2,
				},
			},
		},
		{
			name:   "property refs",
			source: map[string]any{"a": *mockPropertRef1, "b": 2},
			target: map[string]any{"a": *mockPropertRef2, "b": 2},
			expected: map[string]ValuePair{
				"a": {
					Source: *mockPropertRef1,
					Target: *mockPropertRef2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeNestedDiffs(tt.source, tt.target)
			assertValuePairsEqual(t, tt.expected, result)
		})
	}
}

func TestComputeNestedDiffs_Arrays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   any
		target   any
		expected map[string]ValuePair
	}{
		{
			name:   "single element changed",
			source: []any{1, 2, 3},
			target: []any{1, 5, 3},
			expected: map[string]ValuePair{
				"1": {Source: 2, Target: 5},
			},
		},
		{
			name:   "multiple elements changed",
			source: []any{1, 2, 3},
			target: []any{10, 2, 30},
			expected: map[string]ValuePair{
				"0": {Source: 1, Target: 10},
				"2": {Source: 3, Target: 30},
			},
		},
		{
			name:   "array reordered (all indices different)",
			source: []any{1, 2},
			target: []any{2, 1},
			expected: map[string]ValuePair{
				"0": {Source: 1, Target: 2},
				"1": {Source: 2, Target: 1},
			},
		},
		{
			name:   "new indices added (expand array)",
			source: []any{1},
			target: []any{1, 2},
			expected: map[string]ValuePair{
				"1": {Source: nil, Target: 2},
			},
		},
		{
			name:   "indices removed (shrink array)",
			source: []any{1, 2},
			target: []any{1},
			expected: map[string]ValuePair{
				"1": {Source: 2, Target: nil},
			},
		},
		{
			name:     "empty arrays",
			source:   []any{},
			target:   []any{},
			expected: map[string]ValuePair{},
		},
		{
			name:     "no changes (empty result)",
			source:   []any{1, 2, 3},
			target:   []any{1, 2, 3},
			expected: map[string]ValuePair{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeNestedDiffs(tt.source, tt.target)
			assertValuePairsEqual(t, tt.expected, result)
		})
	}
}

func TestComputeNestedDiffs_Mixed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   any
		target   any
		expected map[string]ValuePair
	}{
		{
			name: "maps containing arrays",
			source: map[string]any{
				"items": []any{1, 2},
			},
			target: map[string]any{
				"items": []any{1, 3},
			},
			expected: map[string]ValuePair{
				"items.1": {Source: 2, Target: 3},
			},
		},
		{
			name: "arrays containing maps",
			source: []any{
				map[string]any{"a": 1},
			},
			target: []any{
				map[string]any{"a": 2},
			},
			expected: map[string]ValuePair{
				"0.a": {Source: 1, Target: 2},
			},
		},
		{
			name: "deep mixed nesting",
			source: map[string]any{
				"x": map[string]any{
					"items": []any{
						map[string]any{"y": 1},
					},
				},
			},
			target: map[string]any{
				"x": map[string]any{
					"items": []any{
						map[string]any{"y": 2},
					},
				},
			},
			expected: map[string]ValuePair{
				"x.items.0.y": {Source: 1, Target: 2},
			},
		},
		{
			name: "multiple array items with nested maps",
			source: map[string]any{
				"servers": []any{
					map[string]any{"host": "a.com", "port": 80},
					map[string]any{"host": "b.com", "port": 443},
				},
			},
			target: map[string]any{
				"servers": []any{
					map[string]any{"host": "a.com", "port": 80},
					map[string]any{"host": "b.com", "port": 8443},
				},
			},
			expected: map[string]ValuePair{
				"servers.1.port": {Source: 443, Target: 8443},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeNestedDiffs(tt.source, tt.target)
			assertValuePairsEqual(t, tt.expected, result)
		})
	}
}

func TestComputeNestedDiffs_TypeChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   any
		target   any
		expected map[string]ValuePair
	}{
		{
			name:   "primitive to map",
			source: 42,
			target: map[string]any{"a": 1},
			expected: map[string]ValuePair{
				"": {Source: 42, Target: map[string]any{"a": 1}},
			},
		},
		{
			name:   "map to primitive",
			source: map[string]any{"a": 1},
			target: 42,
			expected: map[string]ValuePair{
				"": {Source: map[string]any{"a": 1}, Target: 42},
			},
		},
		{
			name:   "array to map",
			source: []any{1, 2},
			target: map[string]any{"a": 1},
			expected: map[string]ValuePair{
				"": {Source: []any{1, 2}, Target: map[string]any{"a": 1}},
			},
		},
		{
			name:   "map to array",
			source: map[string]any{"a": 1},
			target: []any{1, 2},
			expected: map[string]ValuePair{
				"": {Source: map[string]any{"a": 1}, Target: []any{1, 2}},
			},
		},
		{
			name:   "primitive to array",
			source: 42,
			target: []any{1, 2},
			expected: map[string]ValuePair{
				"": {Source: 42, Target: []any{1, 2}},
			},
		},
		{
			name:   "array to primitive",
			source: []any{1, 2},
			target: 42,
			expected: map[string]ValuePair{
				"": {Source: []any{1, 2}, Target: 42},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeNestedDiffs(tt.source, tt.target)
			assertValuePairsEqual(t, tt.expected, result)
		})
	}
}

func TestComputeNestedDiffs_NilHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   any
		target   any
		expected map[string]ValuePair
	}{
		{
			name:     "both nil (empty result)",
			source:   nil,
			target:   nil,
			expected: map[string]ValuePair{},
		},
		{
			name:   "source nil, target has value",
			source: nil,
			target: map[string]any{"a": 1},
			expected: map[string]ValuePair{
				"": {Source: nil, Target: map[string]any{"a": 1}},
			},
		},
		{
			name:   "source has value, target nil",
			source: map[string]any{"a": 1},
			target: nil,
			expected: map[string]ValuePair{
				"": {Source: map[string]any{"a": 1}, Target: nil},
			},
		},
		{
			name: "nested nil values in map",
			source: map[string]any{
				"a": nil,
				"b": 2,
			},
			target: map[string]any{
				"a": 1,
				"b": 2,
			},
			expected: map[string]ValuePair{
				"a": {Source: nil, Target: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeNestedDiffs(tt.source, tt.target)
			assertValuePairsEqual(t, tt.expected, result)
		})
	}
}

func TestComputeNestedDiffs_Primitives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   any
		target   any
		expected map[string]ValuePair
	}{
		{
			name:   "strings changed",
			source: "old",
			target: "new",
			expected: map[string]ValuePair{
				"": {Source: "old", Target: "new"},
			},
		},
		{
			name:   "integers changed",
			source: 10,
			target: 20,
			expected: map[string]ValuePair{
				"": {Source: 10, Target: 20},
			},
		},
		{
			name:   "floats changed",
			source: 1.5,
			target: 2.5,
			expected: map[string]ValuePair{
				"": {Source: 1.5, Target: 2.5},
			},
		},
		{
			name:   "booleans changed",
			source: true,
			target: false,
			expected: map[string]ValuePair{
				"": {Source: true, Target: false},
			},
		},
		{
			name:     "no change (empty result) - string",
			source:   "same",
			target:   "same",
			expected: map[string]ValuePair{},
		},
		{
			name:     "no change (empty result) - int",
			source:   42,
			target:   42,
			expected: map[string]ValuePair{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeNestedDiffs(tt.source, tt.target)
			assertValuePairsEqual(t, tt.expected, result)
		})
	}
}

func TestComputeNestedDiffs_MultipleArrayElementsChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   any
		target   any
		expected map[string]ValuePair
	}{
		{
			name: "multiple rules changed in array",
			source: map[string]any{
				"rules": []any{
					map[string]any{"id": "rule1", "enabled": true, "value": 10},
					map[string]any{"id": "rule2", "enabled": true, "value": 20},
					map[string]any{"id": "rule3", "enabled": true, "value": 30},
				},
			},
			target: map[string]any{
				"rules": []any{
					map[string]any{"id": "rule1", "enabled": false, "value": 15},
					map[string]any{"id": "rule2", "enabled": false, "value": 25},
					map[string]any{"id": "rule3", "enabled": false, "value": 35},
				},
			},
			expected: map[string]ValuePair{
				"rules.0.enabled": {Source: true, Target: false},
				"rules.0.value":   {Source: 10, Target: 15},
				"rules.1.enabled": {Source: true, Target: false},
				"rules.1.value":   {Source: 20, Target: 25},
				"rules.2.enabled": {Source: true, Target: false},
				"rules.2.value":   {Source: 30, Target: 35},
			},
		},
		{
			name: "changes in first and middle rules only",
			source: map[string]any{
				"rules": []any{
					map[string]any{"id": "rule1", "enabled": true},
					map[string]any{"id": "rule2", "enabled": true},
					map[string]any{"id": "rule3", "enabled": true},
				},
			},
			target: map[string]any{
				"rules": []any{
					map[string]any{"id": "rule1", "enabled": false},
					map[string]any{"id": "rule2", "enabled": false},
					map[string]any{"id": "rule3", "enabled": true},
				},
			},
			expected: map[string]ValuePair{
				"rules.0.enabled": {Source: true, Target: false},
				"rules.1.enabled": {Source: true, Target: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeNestedDiffs(tt.source, tt.target)
			assertValuePairsEqual(t, tt.expected, result)
		})
	}
}

// assertValuePairsEqual compares two maps of ValuePair, handling PropertyRef values specially
// since they contain function fields that cannot be compared with reflect.DeepEqual
func assertValuePairsEqual(t *testing.T, expected, actual map[string]ValuePair) {
	t.Helper()

	// Check that both maps have the same keys
	assert.Equal(t, len(expected), len(actual), "maps should have the same number of keys")

	for key, expectedPair := range expected {
		actualPair, exists := actual[key]
		assert.True(t, exists, "key %q should exist in actual", key)

		// Compare Source values
		assertValuesEqual(t, expectedPair.Source, actualPair.Source, "Source value for key %q", key)

		// Compare Target values
		assertValuesEqual(t, expectedPair.Target, actualPair.Target, "Target value for key %q", key)
	}
}

// assertValuesEqual compares two values, handling PropertyRef specially
func assertValuesEqual(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()

	// Handle PropertyRef values (not pointers)
	if expectedRef, ok := expected.(resources.PropertyRef); ok {
		actualRef, ok := actual.(resources.PropertyRef)
		assert.True(t, ok, msgAndArgs...)
		assertPropertyRefEqual(t, &expectedRef, &actualRef)
		return
	}

	// Handle PropertyRef pointers
	if expectedRef, ok := expected.(*resources.PropertyRef); ok {
		actualRef, ok := actual.(*resources.PropertyRef)
		assert.True(t, ok, msgAndArgs...)
		assertPropertyRefEqual(t, expectedRef, actualRef)
		return
	}

	// For all other types, use reflect.DeepEqual
	assert.True(t, reflect.DeepEqual(expected, actual), msgAndArgs...)
}

func assertPropertyRefEqual(t *testing.T, expected, actual *resources.PropertyRef) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}
	assert.NotNil(t, expected, "expected PropertyRef should not be nil")
	assert.NotNil(t, actual, "actual PropertyRef should not be nil")
	assert.Equal(t, expected.URN, actual.URN, "PropertyRef URN should be equal")
	assert.Equal(t, expected.Property, actual.Property, "PropertyRef Property should be equal")
	assert.Equal(t, expected.IsResolved, actual.IsResolved, "PropertyRef IsResolved should be equal")
	assert.Equal(t, expected.Value, actual.Value, "PropertyRef Value should be equal")
}
