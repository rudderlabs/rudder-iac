package telemetry

import (
	"encoding/json"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
)

const (
	CommandExecutedEvent = "CLI Command Executed"
)

type KV struct {
	K string
	V interface{}
}

func TrackCommand(command string, err error, extras ...KV) {

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

	if err := telemetry.TrackEvent(CommandExecutedEvent, props); err != nil {
		log.Error("failed to track command", "error", err)
	}
}
