package telemetry

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/rudderlabs/analytics-go/v4"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

var (
	once  sync.Once
	v     string
	ready atomic.Bool
)

// Ready reports whether Initialise ran with telemetry enabled, which is the
// point where the config's opt-out setting has been read.
func Ready() bool { return ready.Load() }

func Initialise(version string) {
	once.Do(func() {

		if config.GetConfig().Telemetry.Disabled {
			return
		}

		v = version
		ready.Store(true)

		if config.GetConfig().Telemetry.AnonymousID == "" {
			anonymousID := uuid.New().String()
			config.SetTelemetryAnonymousID(anonymousID)
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
		anonymousID  = config.GetConfig().Telemetry.AnonymousID
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
		Event:       event,
		Properties:  properties,
		AnonymousId: anonymousID,
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
