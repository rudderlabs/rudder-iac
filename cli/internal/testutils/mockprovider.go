package testutils

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// MockProvider is a mock implementation of the project.Provider interface for testing.
type MockProvider struct {
	SupportedKinds         []string
	SupportedTypes         []string
	ValidateErr            error
	LoadSpecErr            error
	GetResourceGraphVal    *resources.Graph
	GetResourceGraphErr    error
	LoadStateVal           *state.State
	LoadStateErr           error
	PutResourceStateErr    error
	DeleteResourceStateErr error
	CreateVal              *resources.ResourceData
	CreateErr              error
	UpdateVal              *resources.ResourceData
	UpdateErr              error
	DeleteErr              error

	// Tracking calls
	ValidateCalledCount              int
	LoadSpecCalledWithArgs           []LoadSpecArgs
	GetResourceGraphCalledCount      int
	LoadStateCalledCount             int
	PutResourceStateCalledWithArg    PutResourceStateArgs
	DeleteResourceStateCalledWithArg *state.ResourceState
	CreateCalledWithArg              CreateArgs
	UpdateCalledWithArg              UpdateArgs
	DeleteCalledWithArg              DeleteArgs
}

// LoadSpecArgs stores arguments for LoadSpec calls
type LoadSpecArgs struct {
	Path string
	Spec *specs.Spec
}

// PutResourceStateArgs stores arguments for PutResourceState calls
type PutResourceStateArgs struct {
	URN   string
	State *state.ResourceState
}

// CreateArgs stores arguments for Create calls
type CreateArgs struct {
	ID           string
	ResourceType string
	Data         resources.ResourceData
}

// UpdateArgs stores arguments for Update calls
type UpdateArgs struct {
	ID           string
	ResourceType string
	Data         resources.ResourceData
	State        resources.ResourceData
}

// DeleteArgs stores arguments for Delete calls
type DeleteArgs struct {
	ID           string
	ResourceType string
	State        resources.ResourceData
}

// NewMockProvider creates a new MockProvider with initialized tracking fields.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		LoadSpecCalledWithArgs: make([]LoadSpecArgs, 0),
	}
}

func (m *MockProvider) GetSupportedKinds() []string {
	return m.SupportedKinds
}

func (m *MockProvider) GetSupportedTypes() []string {
	return m.SupportedTypes
}

func (m *MockProvider) Validate() error {
	m.ValidateCalledCount++
	return m.ValidateErr
}

func (m *MockProvider) LoadSpec(path string, s *specs.Spec) error {
	m.LoadSpecCalledWithArgs = append(m.LoadSpecCalledWithArgs, LoadSpecArgs{Path: path, Spec: s})
	return m.LoadSpecErr
}

func (m *MockProvider) GetResourceGraph() (*resources.Graph, error) {
	m.GetResourceGraphCalledCount++
	return m.GetResourceGraphVal, m.GetResourceGraphErr
}

func (m *MockProvider) LoadState(ctx context.Context) (*state.State, error) {
	m.LoadStateCalledCount++
	return m.LoadStateVal, m.LoadStateErr
}

func (m *MockProvider) PutResourceState(ctx context.Context, URN string, s *state.ResourceState) error {
	m.PutResourceStateCalledWithArg = PutResourceStateArgs{URN: URN, State: s}
	return m.PutResourceStateErr
}

func (m *MockProvider) DeleteResourceState(ctx context.Context, s *state.ResourceState) error {
	m.DeleteResourceStateCalledWithArg = s
	return m.DeleteResourceStateErr
}

func (m *MockProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	m.CreateCalledWithArg = CreateArgs{ID: ID, ResourceType: resourceType, Data: data}
	return m.CreateVal, m.CreateErr
}

func (m *MockProvider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, s resources.ResourceData) (*resources.ResourceData, error) {
	m.UpdateCalledWithArg = UpdateArgs{ID: ID, ResourceType: resourceType, Data: data, State: s}
	return m.UpdateVal, m.UpdateErr
}

func (m *MockProvider) Delete(ctx context.Context, ID string, resourceType string, s resources.ResourceData) error {
	m.DeleteCalledWithArg = DeleteArgs{ID: ID, ResourceType: resourceType, State: s}
	return m.DeleteErr
}

// ResetCallCounters resets all call counters and argument trackers.
func (m *MockProvider) ResetCallCounters() {
	m.ValidateCalledCount = 0
	m.LoadSpecCalledWithArgs = make([]LoadSpecArgs, 0)
	m.GetResourceGraphCalledCount = 0
	m.LoadStateCalledCount = 0
	m.PutResourceStateCalledWithArg = PutResourceStateArgs{}
	m.DeleteResourceStateCalledWithArg = nil
	m.CreateCalledWithArg = CreateArgs{}
	m.UpdateCalledWithArg = UpdateArgs{}
	m.DeleteCalledWithArg = DeleteArgs{}
}
