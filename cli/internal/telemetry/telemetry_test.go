package telemetry

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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

func setWriteKey(t *testing.T, key string) {
	t.Helper()
	was := viper.Get("telemetry.writeKey")
	viper.Set("telemetry.writeKey", key)
	t.Cleanup(func() { viper.Set("telemetry.writeKey", was) })
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

	setWriteKey(t, "test-write-key")

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

// A build from source has no write key. It must not print the notice, which
// promises collection, or become ready to track.
func TestInitialiseDoesNothingWithoutAWriteKey(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(cfgFile, []byte("{}"), 0o644))
	was := viper.ConfigFileUsed()
	viper.SetConfigFile(cfgFile)
	t.Cleanup(func() { viper.SetConfigFile(was) })

	setTelemetry(t, false, "")
	setWriteKey(t, "")

	var out bytes.Buffer
	noticeOut = &out
	t.Cleanup(func() { noticeOut = os.Stderr })

	// ready is process-wide and the test above leaves it set.
	wasReady := ready.Swap(false)
	t.Cleanup(func() { ready.Store(wasReady) })

	initialise("test")

	assert.False(t, Ready())
	assert.Empty(t, out.String())

	saved, err := os.ReadFile(cfgFile)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(saved))
}

func TestTrackSendsOnlyWithAWriteKey(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
	}))
	t.Cleanup(srv.Close)

	was := viper.Get("telemetry.dataplaneURL")
	viper.Set("telemetry.dataplaneURL", srv.URL)
	t.Cleanup(func() { viper.Set("telemetry.dataplaneURL", was) })
	setTelemetry(t, false, "test-anonymous-id")

	t.Run("no write key sends nothing", func(t *testing.T) {
		setWriteKey(t, "")
		requests = 0

		require.NoError(t, TrackEvent("e", nil))
		assert.Zero(t, requests)
	})

	t.Run("a write key sends the event", func(t *testing.T) {
		setWriteKey(t, "test-write-key")
		requests = 0

		require.NoError(t, TrackEvent("e", nil))
		assert.Positive(t, requests)
	})
}
