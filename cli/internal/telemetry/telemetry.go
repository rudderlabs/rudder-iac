package telemetry

import (
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/rudderlabs/analytics-go/v4"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

var (
	once              sync.Once
	telemetryDisabled bool
)

func getEnvWithFallback(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Initialise() {
	once.Do(func() {
		conf := config.GetConfig()

		if conf.Telemetry.Disabled {
			telemetryDisabled = true
			return
		}

		disabled := getEnvWithFallback("RUDDER_CLI_TELEMETRY_DISABLED", "")
		if disabled != "" && disabled == "1" {
			telemetryDisabled = true
			return
		}

		if conf.Telemetry.UserID == "" {
			userID := uuid.New().String()
			config.SetTelemetryUserID(userID)
		}
	})
}

func DisableTelemetry() {
	telemetryDisabled := true
	config.SetTelemetryDisabled(telemetryDisabled)
}

func EnableTelemetry() {
	telemetryDisabled := false
	config.SetTelemetryDisabled(telemetryDisabled)
}

func track(event string, properties analytics.Properties) error {
	if telemetryDisabled {
		return nil
	}

	var (
		writeKey     = config.GetTelemetryWriteKey()
		dataplaneURL = config.GetTelemetryDataplaneURL()
	)

	client := analytics.New(writeKey, dataplaneURL)
	defer client.Close()

	userID := config.GetTelemetryUserID()
	return client.Enqueue(analytics.Track{
		Event:      event,
		UserId:     userID,
		Properties: properties,
	})
}
