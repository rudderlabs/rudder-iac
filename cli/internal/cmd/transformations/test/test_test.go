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

	// Create a file for the "not a directory" case; t.TempDir() cleans up automatically.
	notADir := filepath.Join(t.TempDir(), "not-a-dir.json")
	f, err := os.Create(notADir)
	require.NoError(t, err)
	f.Close()

	tests := []struct {
		name          string
		args          []string
		all           bool
		modified      bool
		outputDir     string
		expectedError bool
		errorContains string
	}{
		// Valid cases
		{
			name:          "valid single ID",
			args:          []string{"my-transformation"},
			all:           false,
			modified:      false,
			expectedError: false,
		},
		{
			name:          "valid --all flag",
			args:          []string{},
			all:           true,
			modified:      false,
			expectedError: false,
		},
		{
			name:          "valid --modified flag",
			args:          []string{},
			all:           false,
			modified:      true,
			expectedError: false,
		},
		{
			name:          "valid --output-dir with existing dir",
			args:          []string{},
			all:           true,
			modified:      false,
			outputDir:     t.TempDir(),
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
			name:          "invalid --output-dir with non-existent dir",
			args:          []string{},
			all:           true,
			modified:      false,
			outputDir:     "/nonexistent/dir",
			expectedError: true,
			errorContains: "output-dir does not exist",
		},
		{
			name:          "invalid --output-dir pointing to a file",
			args:          []string{},
			all:           true,
			modified:      false,
			outputDir:     notADir,
			expectedError: true,
			errorContains: "output-dir is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateFlags(tt.args, tt.all, tt.modified, tt.outputDir)

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
	t.Run("writes JSON to test-results/results.json", func(t *testing.T) {
		dir := t.TempDir()

		results := &testorchestrator.TestResults{Status: testorchestrator.RunStatusExecuted}
		err := writeResultsFile(dir, results)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(dir, "test-results", "results.json"))
		require.NoError(t, err)

		var got testorchestrator.TestResults
		require.NoError(t, json.Unmarshal(data, &got))
		assert.Equal(t, testorchestrator.RunStatusExecuted, got.Status)
	})

	t.Run("creates test-results dir if not exists", func(t *testing.T) {
		dir := t.TempDir()

		err := writeResultsFile(dir, &testorchestrator.TestResults{Status: testorchestrator.RunStatusExecuted})
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(dir, "test-results"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("silently overwrites existing results.json", func(t *testing.T) {
		dir := t.TempDir()

		first := &testorchestrator.TestResults{Status: testorchestrator.RunStatusExecuted}
		require.NoError(t, writeResultsFile(dir, first))

		second := &testorchestrator.TestResults{Status: testorchestrator.RunStatusNoResources}
		require.NoError(t, writeResultsFile(dir, second))

		data, err := os.ReadFile(filepath.Join(dir, "test-results", "results.json"))
		require.NoError(t, err)

		var got testorchestrator.TestResults
		require.NoError(t, json.Unmarshal(data, &got))
		assert.Equal(t, testorchestrator.RunStatusNoResources, got.Status)
	})

	t.Run("returns error when dir is not writable", func(t *testing.T) {
		err := writeResultsFile("/nonexistent/dir", &testorchestrator.TestResults{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating test-results directory")
	})
}
