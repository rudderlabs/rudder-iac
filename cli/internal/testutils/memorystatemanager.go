package testutils

import (
	"context"
	"encoding/json"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// MemoryStateManager is a simple in-memory implementation of the StateManager interface.
// It serializes the state to a JSON string and stores it in memory, in order to emulate
// the behavior of a persistent state store which would have to persist a JSON instead of a Go struct.
type MemoryStateManager struct {
	json json.RawMessage
	// SaveLog is a log of all the states, in JSON format, that have been saved to the manager.
	// This can be used in tests to check how many times the state has been saved and what the state was at each point.
	SaveLog []json.RawMessage
}

func (m *MemoryStateManager) Load(_ context.Context) (*state.State, error) {
	s, err := state.FromJSON(m.json)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (m *MemoryStateManager) Save(_ context.Context, s *state.State) error {
	json, err := state.ToJSON(s)
	if err != nil {
		return err
	}
	m.json = json
	m.SaveLog = append(m.SaveLog, json)
	return nil
}
