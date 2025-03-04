package telemetry

import (
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/rudderlabs/analytics-go/v4"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

var (
	once              sync.Once
	v                 string
	telemetryDisabled bool
)

func getEnvWithFallback(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Initialise(version string) {
	once.Do(func() {
		conf := config.GetConfig()

		v = version
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

	client, err := analytics.NewWithConfig(writeKey, analytics.Config{
		DataPlaneUrl: dataplaneURL,
		Logger:       NewTelemetryLogger(),
	})

	if err != nil {
		return fmt.Errorf("failed to create analytics client: %w", err)
	}

	defer client.Close()

	userID := config.GetTelemetryUserID()
	return client.Enqueue(analytics.Track{
		Event:      event,
		Properties: properties,
		UserId:     userID,
		Context: &analytics.Context{
			App: analytics.AppInfo{
				Name:    "rudder-cli",
				Version: v,
			},
		},
	})
}

func TrackEvent(event string, props map[string]interface{}) error {
	properties := analytics.NewProperties()

	for k, v := range props {
		properties.Set(k, v)
	}

	return track(event, properties)
}
