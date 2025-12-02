package telemetry

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

var (
	once sync.Once
	tel  *telemetry
)

const (
	CommandExecutedEvent = "CLI Command Executed"
)

type KV struct {
	K string
	V any
}

func Initialise(version string) {
	once.Do(func() {
		if config.GetConfig().Telemetry.Disabled {
			return
		}

		writeKey := config.GetConfig().Telemetry.WriteKey
		dataplaneURL := config.GetConfig().Telemetry.DataplaneURL
		anonymousID := config.GetConfig().Telemetry.AnonymousID

		if anonymousID == "" {
			anonymousID = uuid.New().String()
			config.SetTelemetryAnonymousID(anonymousID)
		}

		tc, err := newTelemetry(version, anonymousID, writeKey, dataplaneURL)
		if err != nil {
			log.Errorf("failed to initialize telemetry: %v", err)
			return
		}

		tel = tc
	})
}

func TrackCommand(command string, err error, extras ...KV) {
	if tel == nil {
		return
	}

	props := map[string]interface{}{
		"command": command,
		"errored": err != nil,
	}

	for _, extra := range extras {
		props[extra.K] = extra.V
	}

	// Automatically add experimental flags
	cfg := config.GetConfig()
	experimentalData, _ := json.Marshal(cfg.ExperimentalFlags)
	var experimental map[string]interface{}
	json.Unmarshal(experimentalData, &experimental)
	props["experimental"] = experimental

	if err := tel.Track(CommandExecutedEvent, props); err != nil {
		log.Errorf("failed to track command: %v", err)
	}
}
