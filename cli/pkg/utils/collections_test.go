package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestItem struct {
	ID   string
	Name string
}

type IntItem struct {
	Value int
}

func TestHasDuplicates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		items          []string
		expectedResult bool
	}{
		{
			name:           "NoDuplicates",
			items:          []string{"a", "b", "c"},
			expectedResult: false,
		},
		{
			name:           "WithDuplicates",
			items:          []string{"a", "b", "a"},
			expectedResult: true,
		},
		{
			name:           "EmptySlice",
			items:          []string{},
			expectedResult: false,
		},
		{
			name:           "SingleItem",
			items:          []string{"a"},
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := HasDuplicates(tc.items)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestFindDuplicates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		items              []string
		expectedDuplicates []string
	}{
		{
			name:               "NoDuplicates",
			items:              []string{"a", "b", "c"},
			expectedDuplicates: []string{},
		},
		{
			name:               "WithDuplicates",
			items:              []string{"a", "b", "a", "c", "b"},
			expectedDuplicates: []string{"a", "b"},
		},
		{
			name:               "EmptySlice",
			items:              []string{},
			expectedDuplicates: []string{},
		},
		{
			name:               "AllSame",
			items:              []string{"a", "a", "a"},
			expectedDuplicates: []string{"a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			duplicates := FindDuplicates(tc.items)
			assert.ElementsMatch(t, tc.expectedDuplicates, duplicates)
		})
	}
}

func TestCheckDuplicatesWithReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		items          []TestItem
		expectedErrors int
	}{
		{
			name: "NoDuplicates",
			items: []TestItem{
				{ID: "1", Name: "Alice"},
				{ID: "2", Name: "Bob"},
			},
			expectedErrors: 0,
		},
		{
			name: "WithDuplicateIDs",
			items: []TestItem{
				{ID: "1", Name: "Alice"},
				{ID: "2", Name: "Bob"},
				{ID: "1", Name: "Charlie"}, // Duplicate ID
			},
			expectedErrors: 1,
		},
		{
			name:           "EmptySlice",
			items:          []TestItem{},
			expectedErrors: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			keyFn := func(item TestItem) string { return item.ID }
			referenceFn := func(item TestItem) string { return item.Name }

			errors := CheckDuplicatesWithReference(tc.items, keyFn, referenceFn, "test")

			assert.Len(t, errors, tc.expectedErrors)

			if tc.expectedErrors > 0 {
				// Verify error structure
				assert.Equal(t, "1", errors[0].Key)
				assert.Equal(t, "Alice", errors[0].FirstRef)
				assert.Equal(t, "Charlie", errors[0].DuplicateRef)
				assert.Contains(t, errors[0].ErrorMessage, "duplicate")
			}
		})
	}
}

func TestCheckDuplicates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		items       []TestItem
		expectError bool
	}{
		{
			name: "NoDuplicates",
			items: []TestItem{
				{ID: "1", Name: "Alice"},
				{ID: "2", Name: "Bob"},
			},
			expectError: false,
		},
		{
			name: "WithDuplicates",
			items: []TestItem{
				{ID: "1", Name: "Alice"},
				{ID: "1", Name: "Bob"}, // Duplicate ID
			},
			expectError: true,
		},
		{
			name:        "EmptySlice",
			items:       []TestItem{},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			keyFn := func(item TestItem) string { return item.ID }
			err := CheckDuplicates(tc.items, keyFn, "test")

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "duplicate")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckDuplicatesWithReferenceComplex(t *testing.T) {
	t.Parallel()

	// Test with more complex scenarios
	tests := []struct {
		name               string
		items              []TestItem
		expectedDuplicates int
	}{
		{
			name: "MultipleDuplicates",
			items: []TestItem{
				{ID: "1", Name: "Alice"},
				{ID: "2", Name: "Bob"},
				{ID: "1", Name: "Charlie"}, // Duplicate ID=1
				{ID: "2", Name: "David"},   // Duplicate ID=2
			},
			expectedDuplicates: 2,
		},
		{
			name: "SingleDuplicate",
			items: []TestItem{
				{ID: "1", Name: "Alice"},
				{ID: "2", Name: "Bob"},
				{ID: "1", Name: "Charlie"}, // Duplicate ID=1
			},
			expectedDuplicates: 1,
		},
		{
			name: "NoOverlap",
			items: []TestItem{
				{ID: "1", Name: "Alice"},
				{ID: "2", Name: "Bob"},
				{ID: "3", Name: "Charlie"},
			},
			expectedDuplicates: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			keyFn := func(item TestItem) string { return item.ID }
			referenceFn := func(item TestItem) string { return item.Name }

			errors := CheckDuplicatesWithReference(tc.items, keyFn, referenceFn, "test")
			assert.Len(t, errors, tc.expectedDuplicates)
		})
	}
}

// Test helper functions for collections that work with custom types
func TestHasDuplicatesWithTransformation(t *testing.T) {
	t.Parallel()

	// Test with TestItem using transformation to string keys
	testItems := []TestItem{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
		{ID: "1", Name: "Charlie"}, // Duplicate ID
	}

	// Transform to IDs and check for duplicates
	ids := make([]string, len(testItems))
	for i, item := range testItems {
		ids[i] = item.ID
	}

	result := HasDuplicates(ids)
	assert.True(t, result, "Should detect duplicate IDs")

	// Transform to Names and check for duplicates
	names := make([]string, len(testItems))
	for i, item := range testItems {
		names[i] = item.Name
	}

	result = HasDuplicates(names)
	assert.False(t, result, "Should not detect duplicate names")
}

func TestFindDuplicatesWithTransformation(t *testing.T) {
	t.Parallel()

	// Test with TestItem using transformation to string keys
	testItems := []TestItem{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
		{ID: "1", Name: "Charlie"}, // Duplicate ID
		{ID: "3", Name: "Alice"},   // Duplicate Name
	}

	// Transform to IDs and find duplicates
	ids := make([]string, len(testItems))
	for i, item := range testItems {
		ids[i] = item.ID
	}

	duplicateIDs := FindDuplicates(ids)
	assert.ElementsMatch(t, []string{"1"}, duplicateIDs)

	// Transform to Names and find duplicates
	names := make([]string, len(testItems))
	for i, item := range testItems {
		names[i] = item.Name
	}

	duplicateNames := FindDuplicates(names)
	assert.ElementsMatch(t, []string{"Alice"}, duplicateNames)
}

func TestIntItemTransformation(t *testing.T) {
	t.Parallel()

	// Test with IntItem using transformation to string keys
	intItems := []IntItem{
		{Value: 1},
		{Value: 2},
		{Value: 1}, // Duplicate
	}

	// Transform to string representations
	stringValues := make([]string, len(intItems))
	for i, item := range intItems {
		stringValues[i] = fmt.Sprintf("%d", item.Value)
	}

	result := HasDuplicates(stringValues)
	assert.True(t, result, "Should detect duplicate values")

	duplicates := FindDuplicates(stringValues)
	assert.ElementsMatch(t, []string{"1"}, duplicates)
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("NilSlice", func(t *testing.T) {
		t.Parallel()

		var nilSlice []string
		assert.False(t, HasDuplicates(nilSlice))
		assert.Empty(t, FindDuplicates(nilSlice))

		var nilTestItems []TestItem
		keyFn := func(item TestItem) string { return item.ID }
		referenceFn := func(item TestItem) string { return item.Name }
		assert.Empty(t, CheckDuplicatesWithReference(nilTestItems, keyFn, referenceFn, "test"))
	})

	t.Run("EmptyStrings", func(t *testing.T) {
		t.Parallel()

		items := []string{"", "", "a"}
		assert.True(t, HasDuplicates(items))
		assert.ElementsMatch(t, []string{""}, FindDuplicates(items))
	})

	t.Run("WhitespaceStrings", func(t *testing.T) {
		t.Parallel()

		items := []string{" ", "  ", " "}
		assert.True(t, HasDuplicates(items))
		assert.ElementsMatch(t, []string{" "}, FindDuplicates(items))
	})

	t.Run("EmptyKeys", func(t *testing.T) {
		t.Parallel()

		// Test that empty keys are skipped
		items := []TestItem{
			{ID: "", Name: "Alice"},
			{ID: "", Name: "Bob"},
			{ID: "1", Name: "Charlie"},
		}

		keyFn := func(item TestItem) string { return item.ID }
		referenceFn := func(item TestItem) string { return item.Name }

		// Should not report duplicates for empty keys
		errors := CheckDuplicatesWithReference(items, keyFn, referenceFn, "test")
		assert.Empty(t, errors)

		err := CheckDuplicates(items, keyFn, "test")
		assert.NoError(t, err)
	})
}
