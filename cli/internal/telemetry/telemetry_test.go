package telemetry

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Initialise runs once per process, so this is its only test. Ready is what
// gates the invalid-input report; without the Store(true) a failure that
// cobra raises would never be sent.
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

func TestInitialiseMarksTelemetryReady(t *testing.T) {
	// A preset id keeps Initialise from writing one into the user's config file.
	setTelemetry(t, false, "test-anonymous-id")

	assert.False(t, Ready())

	Initialise("test")

	assert.True(t, Ready())
}
