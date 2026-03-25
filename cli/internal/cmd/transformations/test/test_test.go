package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
)

func TestValidateFlags(t *testing.T) {
	t.Parallel()

	// Create an existing file for the "already exists" and "force" cases.
	existingFile := filepath.Join(t.TempDir(), "test-results.json")
	f, err := os.Create(existingFile)
	require.NoError(t, err)
	f.Close()

	tests := []struct {
		name          string
		args          []string
		all           bool
		modified      bool
		location      string
		output        string
		force         bool
		expectedError bool
		errorContains string
	}{
		// Valid cases
		{
			name:          "valid single ID",
			args:          []string{"my-transformation"},
			all:           false,
			modified:      false,
			location:      t.TempDir(),
			expectedError: false,
		},
		{
			name:          "valid --all flag",
			args:          []string{},
			all:           true,
			modified:      false,
			location:      t.TempDir(),
			expectedError: false,
		},
		{
			name:          "valid --modified flag",
			args:          []string{},
			all:           false,
			modified:      true,
			location:      t.TempDir(),
			expectedError: false,
		},
		{
			name:          "valid -o with existing dir base",
			args:          []string{},
			all:           true,
			modified:      false,
			output:        filepath.Join(t.TempDir(), "results.json"),
			expectedError: false,
		},
		{
			name:          "valid -o with --force on existing file",
			args:          []string{},
			all:           true,
			modified:      false,
			output:        existingFile,
			force:         true,
			expectedError: false,
		},

		// Invalid cases
		{
			name:          "ID + --all",
			args:          []string{"my-transformation"},
			all:           true,
			modified:      false,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "ID + --modified",
			args:          []string{"my-transformation"},
			all:           false,
			modified:      true,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "--all + --modified",
			args:          []string{},
			all:           true,
			modified:      true,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "all three modes",
			args:          []string{"my-transformation"},
			all:           true,
			modified:      true,
			expectedError: true,
			errorContains: "cannot combine test modes",
		},
		{
			name:          "multiple IDs",
			args:          []string{"transformation-1", "transformation-2"},
			all:           false,
			modified:      false,
			expectedError: true,
			errorContains: "only one transformation/library ID allowed",
		},
		{
			name:          "no mode specified",
			args:          []string{},
			all:           false,
			modified:      false,
			expectedError: true,
			errorContains: "must specify either an ID, --all, or --modified",
		},
		{
			name:          "invalid -o with non-existent base dir",
			args:          []string{},
			all:           true,
			modified:      false,
			output:        "/nonexistent/dir/results.json",
			expectedError: true,
			errorContains: "output directory does not exist",
		},
		{
			name:          "invalid -o file already exists without --force",
			args:          []string{},
			all:           true,
			modified:      false,
			output:        existingFile,
			expectedError: true,
			errorContains: "output file already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateFlags(tt.args, tt.all, tt.modified, tt.location, tt.output, tt.force)

			if tt.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWriteResultsFile(t *testing.T) {
	t.Run("writes JSON to given path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "results.json")

		results := &testorchestrator.TestResults{Status: testorchestrator.RunStatusExecuted}
		err := writeResultsFile(path, results)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var got testorchestrator.TestResults
		require.NoError(t, json.Unmarshal(data, &got))
		assert.Equal(t, testorchestrator.RunStatusExecuted, got.Status)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "results.json")

		first := &testorchestrator.TestResults{Status: testorchestrator.RunStatusExecuted}
		require.NoError(t, writeResultsFile(path, first))

		second := &testorchestrator.TestResults{Status: testorchestrator.RunStatusNoResources}
		require.NoError(t, writeResultsFile(path, second))

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var got testorchestrator.TestResults
		require.NoError(t, json.Unmarshal(data, &got))
		assert.Equal(t, testorchestrator.RunStatusNoResources, got.Status)
	})

	t.Run("returns error when path is not writable", func(t *testing.T) {
		err := writeResultsFile("/nonexistent/dir/results.json", &testorchestrator.TestResults{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating results file")
	})
}
