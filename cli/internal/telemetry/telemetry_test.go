package telemetry

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Initialise runs once per process, so this is its only test. Ready is what
// gates the invalid-input report; without the Store(true) a failure that
// cobra raises would never be sent.
func TestInitialiseMarksTelemetryReady(t *testing.T) {
	viper.Set("telemetry.disabled", false)
	// A preset id keeps Initialise from writing one into the user's config file.
	viper.Set("telemetry.anonymousId", "test-anonymous-id")
	t.Cleanup(func() {
		viper.Set("telemetry.disabled", nil)
		viper.Set("telemetry.anonymousId", nil)
	})

	assert.False(t, Ready())

	Initialise("test")

	assert.True(t, Ready())
}
