package telemetry

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTelemetry sets the two telemetry settings for one test and puts back what
// was there before.
func setTelemetry(t *testing.T, disabled bool, anonymousID string) {
	t.Helper()
	wasDisabled, wasID := viper.Get("telemetry.disabled"), viper.Get("telemetry.anonymousId")
	viper.Set("telemetry.disabled", disabled)
	viper.Set("telemetry.anonymousId", anonymousID)
	t.Cleanup(func() {
		viper.Set("telemetry.disabled", wasDisabled)
		viper.Set("telemetry.anonymousId", wasID)
	})
}

// Initialise runs once per process, so this is its only test. Ready is what
// gates the invalid-input report; without the Store(true) a failure that
// cobra raises would never be sent. An install that has not seen the notice
// gets it once, and the config records that.
func TestInitialiseShowsTheNoticeAndMarksTelemetryReady(t *testing.T) {
	// Initialise writes to the config file, so the test gives it its own.
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(cfgFile, []byte("{}"), 0o644))
	was := viper.ConfigFileUsed()
	viper.SetConfigFile(cfgFile)
	t.Cleanup(func() { viper.SetConfigFile(was) })

	// A preset id keeps the test to the one write it is about.
	setTelemetry(t, false, "test-anonymous-id")

	var out bytes.Buffer
	noticeOut = &out
	t.Cleanup(func() { noticeOut = os.Stderr })

	assert.False(t, Ready())

	Initialise("test")

	assert.True(t, Ready())
	assert.Equal(t, Notice, out.String())

	saved, err := os.ReadFile(cfgFile)
	require.NoError(t, err)
	assert.JSONEq(t, `{"telemetry":{"notice":1}}`, string(saved))
}
