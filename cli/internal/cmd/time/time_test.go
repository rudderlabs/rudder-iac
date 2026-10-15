package timecmd

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdTimePrintsCurrentLocalTime(t *testing.T) {
	fixedTime := time.Date(2026, time.March, 14, 15, 9, 26, 0, time.FixedZone("UTC-7", -7*60*60))
	cmd := newCmdTime(func() time.Time { return fixedTime })
	var output bytes.Buffer
	cmd.SetOut(&output)

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "2026-03-14T15:09:26-07:00\n", output.String())
}

func TestNewCmdTimeMetadata(t *testing.T) {
	cmd := NewCmdTime()

	assert.Equal(t, "time", cmd.Use)
	assert.Equal(t, "Print the current local date and time", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestCmdTimeRejectsArguments(t *testing.T) {
	cmd := newCmdTime(func() time.Time {
		return time.Date(2026, time.March, 14, 15, 9, 26, 0, time.UTC)
	})
	cmd.SetArgs([]string{"unexpected"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}
