package syncer

import (
	"sync"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// sharedState is a thread-safe wrapper around state.State for concurrent access during
// operations execution. Uses sync.RWMutex to prevent race conditions when multiple
// goroutines simultaneously read or write to the state.
type sharedState struct {
	state *state.State
	mutex sync.RWMutex
}

func newSharedState(initialState *state.State) *sharedState {
	return &sharedState{
		state: initialState,
		mutex: sync.RWMutex{},
	}
}

func (s *sharedState) Dereference(data resources.ResourceData) (resources.ResourceData, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return state.Dereference(data, s.state)
}

func (s *sharedState) AddResource(r *state.ResourceState) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.state.AddResource(r)
}

func (s *sharedState) GetResource(urn string) *state.ResourceState {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.state.GetResource(urn)
}

func (s *sharedState) RemoveResource(urn string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.state.RemoveResource(urn)
}
