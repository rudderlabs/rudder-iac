package testutils

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type MemoryStateManager struct {
	s *state.State
}

func (m *MemoryStateManager) Load(_ context.Context) (*state.State, error) {
	return m.s, nil
}

func (m *MemoryStateManager) Save(_ context.Context, s *state.State) error {
	m.s = s
	return nil
}
