package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		all           bool
		modified      bool
		show          bool
		expectedError bool
		errorContains string
	}{
		// Valid cases
		{
			name:          "valid single ID",
			args:          []string{"my-transformation"},
			all:           false,
			modified:      false,
			show:          false,
			expectedError: false,
		},
		{
			name:          "valid --all flag",
			args:          []string{},
			all:           true,
			modified:      false,
			show:          false,
			expectedError: false,
		},
		{
			name:          "valid --modified flag",
			args:          []string{},
			all:           false,
			modified:      true,
			show:          false,
			expectedError: false,
		},
		{
			name:          "valid default-events --show",
			args:          []string{"default-events"},
			all:           false,
			modified:      false,
			show:          true,
			expectedError: false,
		},

		// Invalid cases
		{
			name:          "ID + --all",
			args:          []string{"my-transformation"},
			all:           true,
			modified:      false,
			show:          false,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "ID + --modified",
			args:          []string{"my-transformation"},
			all:           false,
			modified:      true,
			show:          false,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "--all + --modified",
			args:          []string{},
			all:           true,
			modified:      true,
			show:          false,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "all three modes",
			args:          []string{"my-transformation"},
			all:           true,
			modified:      true,
			show:          false,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "multiple IDs",
			args:          []string{"transformation-1", "transformation-2"},
			all:           false,
			modified:      false,
			show:          false,
			expectedError: true,
			errorContains: "only one transformation/library ID allowed",
		},
		{
			name:          "no mode specified",
			args:          []string{},
			all:           false,
			modified:      false,
			show:          false,
			expectedError: true,
			errorContains: "must specify either an ID, --all, or --modified",
		},
		{
			name:          "--show without default-events",
			args:          []string{},
			all:           false,
			modified:      false,
			show:          true,
			expectedError: true,
			errorContains: "--show flag requires 'default-events' argument",
		},
		{
			name:          "--show with wrong argument",
			args:          []string{"my-transformation"},
			all:           false,
			modified:      false,
			show:          true,
			expectedError: true,
			errorContains: "--show flag requires 'default-events' argument",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateFlags(tt.args, tt.all, tt.modified, tt.show)

			if tt.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShowDefaultEvents(t *testing.T) {
	t.Parallel()

	// Test that showDefaultEvents runs without error
	err := showDefaultEvents()
	require.NoError(t, err)
}

func TestGetDefaultEvents(t *testing.T) {
	t.Parallel()

	events := GetDefaultEvents()

	// Verify we have the expected default events
	require.NotEmpty(t, events)
	assert.Contains(t, events, "Track")
	assert.Contains(t, events, "Identify")
	assert.Contains(t, events, "Page")
	assert.Contains(t, events, "Screen")

	// Verify each event has expected structure
	for name, content := range events {
		eventMap, ok := content.(map[string]any)
		assert.True(t, ok, "event %s should be a map[string]any", name)
		assert.NotEmpty(t, eventMap, "event %s should not be empty", name)

		// Verify each event has a type field
		assert.Contains(t, eventMap, "type", "event %s should have a type field", name)
	}
}
