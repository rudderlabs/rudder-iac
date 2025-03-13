package telemetry

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rudderlabs/analytics-go/v4"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

var (
	once sync.Once
	v    string
)

func Initialise(version string) {
	once.Do(func() {

		if config.GetConfig().Telemetry.Disabled {
			return
		}

		v = version

		if config.GetConfig().Telemetry.UserID == "" {
			userID := uuid.New().String()
			config.SetTelemetryUserID(userID)
		}
	})
}

func DisableTelemetry() {
	config.SetTelemetryDisabled(true)
}

func EnableTelemetry() {
	config.SetTelemetryDisabled(false)
}

func track(event string, properties analytics.Properties) error {
	if config.GetConfig().Telemetry.Disabled {
		return nil
	}

	var (
		userID       = config.GetConfig().Telemetry.UserID
		writeKey     = config.GetConfig().Telemetry.WriteKey
		dataplaneURL = config.GetConfig().Telemetry.DataplaneURL
	)

	client, err := analytics.NewWithConfig(writeKey, analytics.Config{
		DataPlaneUrl: dataplaneURL,
		Logger:       NewTelemetryLogger(),
	})

	if err != nil {
		return fmt.Errorf("failed to create analytics client: %w", err)
	}

	defer client.Close()

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
