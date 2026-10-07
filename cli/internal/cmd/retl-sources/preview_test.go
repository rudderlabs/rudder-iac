package retlsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePreviewLimit(t *testing.T) {
	tests := []struct {
		name    string
		limit   int
		wantErr string
	}{
		{name: "lower bound", limit: 1},
		{name: "default", limit: 10},
		{name: "upper bound", limit: 100},
		{name: "zero", limit: 0, wantErr: "--limit must be between 1 and 100, got 0"},
		{name: "negative", limit: -1, wantErr: "--limit must be between 1 and 100, got -1"},
		{name: "above upper bound", limit: 101, wantErr: "--limit must be between 1 and 100, got 101"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePreviewLimit(tt.limit)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

// The limit must be rejected before the project is loaded or any API call is
// made; the nonexistent location proves nothing past the check ran.
func TestPreviewCommandRejectsOutOfRangeLimit(t *testing.T) {
	cmd := newCmdPreview()
	cmd.SetArgs([]string{"my-model", "--limit", "150", "--location", "/nonexistent"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()

	require.Error(t, err)
	assert.EqualError(t, err, "--limit must be between 1 and 100, got 150")
}
