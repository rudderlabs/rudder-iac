package testorchestrator

import (
	_ "embed"
	"encoding/json"
)

//go:embed default_events.json
var defaultEventsJSON []byte

// GetDefaultEvents returns a fresh copy of default test event samples.
// Each call returns a new map to prevent mutation of shared data.
func GetDefaultEvents() map[string]any {
	var events map[string]any
	json.Unmarshal(defaultEventsJSON, &events)
	return events
}
