package testorchestrator

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
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

// ShowDefaultEvents displays the embedded default test events
func ShowDefaultEvents() error {
	events := GetDefaultEvents()

	eventCount := len(events)
	ui.Println(ui.Bold(strings.Repeat("─", 60)))
	ui.Printf("%s\t\t\t\t%d events\n", ui.Bold("DEFAULT TEST EVENTS"), eventCount)
	ui.Println(ui.Bold(strings.Repeat("─", 60)))
	ui.Println()

	for eventType, eventData := range events {
		header := fmt.Sprintf("┌─ %s %s %s",
			strings.Repeat("─", 20),
			ui.Color(eventType, ui.ColorYellow),
			strings.Repeat("─", 20),
		)
		ui.Println(header)

		// Marshal event data to pretty JSON for display
		jsonBytes, err := json.MarshalIndent(eventData, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling event %s to JSON: %w", eventType, err)
		}

		ui.Println(string(jsonBytes))
		ui.Println()
	}

	return nil
}
