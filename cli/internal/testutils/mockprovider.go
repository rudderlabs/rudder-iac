package testutils

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// MockProvider is a mock implementation of the provider.Provider interface for testing.
type MockProvider struct {
	provider.EmptyProvider
	supportedKinds             []string
	supportedTypes             []string
	ValidateArg                *resources.Graph
	ValidateErr                error
	LoadSpecErr                error
	GetResourceGraphVal        *resources.Graph
	GetResourceGraphErr        error
	LoadResourcesFromRemoteVal *resources.RemoteResources
	LoadResourcesFromRemoteErr error
	MapRemoteToStateVal        *state.State
	MapRemoteToStateErr        error
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
	ValidateErrorReturnedCount         int
	LoadSpecCalledWithArgs             []LoadSpecArgs
	ParseSpecCalledWithArgs            []ParseSpecArgs
	GetResourceGraphCalledCount        int
	GetResourceGraphErrorReturnedCount int
	LoadResourcesFromRemoteCalledCount int
	MapRemoteToStateCalledCount        int
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
	RemoteId     string
}

// NewMockProvider creates a new MockProvider with initialized tracking fields.
func NewMockProvider(supportedKinds, supportedTypes []string) *MockProvider {
	return &MockProvider{
		supportedKinds:          supportedKinds,
		supportedTypes:          supportedTypes,
		ParseSpecVal:            &specs.ParsedSpec{ExternalIDs: []string{}},
		LoadSpecCalledWithArgs:  make([]LoadSpecArgs, 0),
		ParseSpecCalledWithArgs: make([]ParseSpecArgs, 0),
	}
}

func (m *MockProvider) SupportedKinds() []string {
	return m.supportedKinds
}

func (m *MockProvider) SupportedTypes() []string {
	return m.supportedTypes
}

func (m *MockProvider) Validate(graph *resources.Graph) error {
	m.ValidateArg = graph
	m.ValidateCalledCount++
	if m.ValidateErr != nil {
		m.ValidateErrorReturnedCount++
	}
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

func (m *MockProvider) ResourceGraph() (*resources.Graph, error) {
	m.GetResourceGraphCalledCount++
	if m.GetResourceGraphErr != nil {
		m.GetResourceGraphErrorReturnedCount++
	}
	return m.GetResourceGraphVal, m.GetResourceGraphErr
}

func (m *MockProvider) LoadResourcesFromRemote(_ context.Context) (*resources.RemoteResources, error) {
	m.LoadResourcesFromRemoteCalledCount++
	return m.LoadResourcesFromRemoteVal, m.LoadResourcesFromRemoteErr
}

func (m *MockProvider) MapRemoteToState(_ *resources.RemoteResources) (*state.State, error) {
	m.MapRemoteToStateCalledCount++
	return m.MapRemoteToStateVal, m.MapRemoteToStateErr
}

func (m *MockProvider) Create(_ context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	m.CreateCalledWithArg = CreateArgs{ID: ID, ResourceType: resourceType, Data: data}
	return m.CreateVal, m.CreateErr
}

func (m *MockProvider) Update(_ context.Context, ID string, resourceType string, data resources.ResourceData, s resources.ResourceData) (*resources.ResourceData, error) {
	m.UpdateCalledWithArg = UpdateArgs{ID: ID, ResourceType: resourceType, Data: data, State: s}
	return m.UpdateVal, m.UpdateErr
}

func (m *MockProvider) Delete(_ context.Context, ID string, resourceType string, s resources.ResourceData) error {
	m.DeleteCalledWithArg = DeleteArgs{ID: ID, ResourceType: resourceType, State: s}
	return m.DeleteErr
}

func (m *MockProvider) Import(_ context.Context, ID string, resourceType string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	m.ImportCalledWithArg = ImportArgs{ID: ID, ResourceType: resourceType, Data: data, RemoteId: remoteId}
	return m.ImportVal, m.ImportErr
}

func (m *MockProvider) LoadImportable(_ context.Context, _ namer.Namer) (*resources.RemoteResources, error) {
	return nil, nil
}

func (m *MockProvider) FormatForExport(collection *resources.RemoteResources, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
	return nil, nil
}

// ResetCallCounters resets all call counters and argument trackers.
func (m *MockProvider) ResetCallCounters() {
	m.ValidateCalledCount = 0
	m.ValidateErrorReturnedCount = 0
	m.LoadSpecCalledWithArgs = make([]LoadSpecArgs, 0)
	m.ParseSpecCalledWithArgs = make([]ParseSpecArgs, 0)
	m.GetResourceGraphCalledCount = 0
	m.GetResourceGraphErrorReturnedCount = 0
	m.CreateCalledWithArg = CreateArgs{}
	m.UpdateCalledWithArg = UpdateArgs{}
	m.DeleteCalledWithArg = DeleteArgs{}
}
