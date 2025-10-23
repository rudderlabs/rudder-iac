package testutils

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// MockProvider is a mock implementation of the project.Provider interface for testing.
type MockProvider struct {
	SupportedKinds             []string
	SupportedTypes             []string
	ValidateArg                *resources.Graph
	ValidateErr                error
	LoadSpecErr                error
	GetResourceGraphVal        *resources.Graph
	GetResourceGraphErr        error
	LoadStateVal               *state.State
	LoadStateErr               error
	LoadResourcesFromRemoteVal *resources.ResourceCollection
	LoadResourcesFromRemoteErr error
	LoadStateFromResourcesVal  *state.State
	LoadStateFromResourcesErr  error
	PutResourceStateErr        error
	DeleteResourceStateErr     error
	CreateVal                  *resources.ResourceData
	CreateErr                  error
	UpdateVal                  *resources.ResourceData
	UpdateErr                  error
	DeleteErr                  error
	ImportVal                  *resources.ResourceData
	ImportErr                  error
	ParseSpecVal               *specs.ParsedSpec
	ParseSpecErr               error

	// Tracking calls
	ValidateCalledCount                int
	LoadSpecCalledWithArgs             []LoadSpecArgs
	ParseSpecCalledWithArgs            []ParseSpecArgs
	GetResourceGraphCalledCount        int
	LoadStateCalledCount               int
	LoadResourcesFromRemoteCalledCount int
	LoadStateFromResourcesCalledCount  int
	PutResourceStateCalledWithArg      PutResourceStateArgs
	DeleteResourceStateCalledWithArg   *state.ResourceState
	CreateCalledWithArg                CreateArgs
	UpdateCalledWithArg                UpdateArgs
	DeleteCalledWithArg                DeleteArgs
	ImportCalledWithArg                ImportArgs
}

// LoadSpecArgs stores arguments for LoadSpec calls
type LoadSpecArgs struct {
	Path string
	Spec *specs.Spec
}

// ParseSpecArgs stores arguments for ParseSpec calls
type ParseSpecArgs struct {
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

// ImportArgs stores arguments for Import calls
type ImportArgs struct {
	ID           string
	ResourceType string
	Data         resources.ResourceData
	WorkspaceId  string
	RemoteId     string
}

// NewMockProvider creates a new MockProvider with initialized tracking fields.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		ParseSpecVal:            &specs.ParsedSpec{ExternalIDs: []string{}},
		LoadSpecCalledWithArgs:  make([]LoadSpecArgs, 0),
		ParseSpecCalledWithArgs: make([]ParseSpecArgs, 0),
	}
}

func (m *MockProvider) GetName() string {
	return "mock"
}

func (m *MockProvider) GetSupportedKinds() []string {
	return m.SupportedKinds
}

func (m *MockProvider) GetSupportedTypes() []string {
	return m.SupportedTypes
}

func (m *MockProvider) Validate(graph *resources.Graph) error {
	m.ValidateArg = graph
	m.ValidateCalledCount++
	return m.ValidateErr
}

func (m *MockProvider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	m.ParseSpecCalledWithArgs = append(m.ParseSpecCalledWithArgs, ParseSpecArgs{Path: path, Spec: s})
	return m.ParseSpecVal, m.ParseSpecErr
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

func (m *MockProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	m.LoadResourcesFromRemoteCalledCount++
	return m.LoadResourcesFromRemoteVal, m.LoadResourcesFromRemoteErr
}

func (m *MockProvider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	m.LoadStateFromResourcesCalledCount++
	return m.LoadStateFromResourcesVal, m.LoadStateFromResourcesErr
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

func (m *MockProvider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	m.ImportCalledWithArg = ImportArgs{ID: ID, ResourceType: resourceType, Data: data, WorkspaceId: workspaceId, RemoteId: remoteId}
	return m.ImportVal, m.ImportErr
}

func (m *MockProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	return nil, nil
}

func (m *MockProvider) FormatForExport(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]importremote.FormattableEntity, error) {
	return nil, nil
}

// ResetCallCounters resets all call counters and argument trackers.
func (m *MockProvider) ResetCallCounters() {
	m.ValidateCalledCount = 0
	m.LoadSpecCalledWithArgs = make([]LoadSpecArgs, 0)
	m.ParseSpecCalledWithArgs = make([]ParseSpecArgs, 0)
	m.GetResourceGraphCalledCount = 0
	m.LoadStateCalledCount = 0
	m.PutResourceStateCalledWithArg = PutResourceStateArgs{}
	m.DeleteResourceStateCalledWithArg = nil
	m.CreateCalledWithArg = CreateArgs{}
	m.UpdateCalledWithArg = UpdateArgs{}
	m.DeleteCalledWithArg = DeleteArgs{}
}
