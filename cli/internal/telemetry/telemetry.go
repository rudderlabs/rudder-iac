package telemetry

import (
	"fmt"
	"io"
	"os"
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

// noticeVersion is the version of Notice. Raise it when what the CLI sends
// changes in a way a user should hear about, and every install is told again.
const noticeVersion = 1

// Notice is printed once per install, before the first event or API call
// that carries telemetry. Version 1 also reaches installs that had telemetry
// on before the telemetry id was tied to a workspace.
const Notice = `rudder-cli sends usage data to RudderStack: the commands you run, their flags,
whether they fail, and the coding agent that runs them. API calls carry a
telemetry id, which ties this usage to your workspace.
Turn it off with 'rudder-cli telemetry disable' or DO_NOT_TRACK=1.
What is sent: https://github.com/rudderlabs/rudder-iac#telemetry
This notice is shown once.
`

// noticeOut is where the notice goes. Standard error keeps it out of output
// that a script or an agent parses.
var noticeOut io.Writer = os.Stderr

func Initialise(version string) {
	once.Do(func() {

		if config.GetConfig().Telemetry.Disabled {
			return
		}

		if config.GetConfig().Telemetry.Notice < noticeVersion {
			fmt.Fprint(noticeOut, Notice)
			config.SetTelemetryNotice(noticeVersion)
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
