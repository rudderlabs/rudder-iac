package telemetry

import "github.com/rudderlabs/rudder-iac/cli/internal/telemetry"

const (
	CommandExecutedEvent = "Command Executed"
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

	if err := telemetry.TrackEvent(CommandExecutedEvent, props); err != nil {
		log.Error("failed to track command", "error", err)
	}
}
