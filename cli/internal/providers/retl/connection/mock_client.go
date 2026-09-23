package connection

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// MockConnectionClient is a recording RETLStore double for the connection
// handler tests. The embedded interface covers the source and preview surfaces
// the handler never touches, so an accidental call panics instead of quietly
// returning a zero value.
//
// Every method refuses a context that is already done, the way a real client
// would: a double that ignores cancellation would hide exactly the bugs the
// import path's detached read-back exists to prevent.
type MockConnectionClient struct {
	retlClient.RETLStore

	CreateCalls        []retlClient.CreateRETLConnectionRequest
	UpdateCalls        []string
	DeleteCalls        []string
	ListCalls          []retlClient.ListRETLConnectionsRequest
	GetCalls           []string
	SetExternalIDCalls []retlClient.SetRETLConnectionExternalIDRequest

	// Sources and Destinations are the two endpoint catalogs eligibility reads;
	// EndpointsErr fails both, and the counters say how often they were read.
	Sources           []retlClient.RETLSource
	Destinations      []apiClient.Destination
	EndpointsErr      error
	SourceListCalls   int
	DestinationsCalls int

	// Sync surface. Syncs is the run history GetSync and ListSyncs serve, and
	// the call slices record what a command asked for.
	Syncs          []retlClient.Sync
	SyncErr        error
	StartSyncCalls []string
	StopSyncCalls  []string
	ListSyncsCalls []retlClient.ListSyncsRequest
	GetSyncCalls   []string

	CreateFunc        func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	UpdateFunc        func(id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	DeleteFunc        func(id string) error
	ListFunc          func(req *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error)
	GetFunc           func(id string) (*retlClient.RETLConnection, error)
	SetExternalIDFunc func(req *retlClient.SetRETLConnectionExternalIDRequest) error
}

var _ retlClient.RETLStore = (*MockConnectionClient)(nil)

func (m *MockConnectionClient) CreateConnection(_ context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	m.CreateCalls = append(m.CreateCalls, *req)
	if m.CreateFunc != nil {
		return m.CreateFunc(req)
	}
	return echoCreated("conn-remote-1", req), nil
}

// echoCreated is the connection the API returns for a create: the request's
// config echoed back under a server id. Create reads it back through
// configFromRemote, so a bare id-only response would look unrepresentable.
func echoCreated(id string, req *retlClient.CreateRETLConnectionRequest) *retlClient.RETLConnection {
	return &retlClient.RETLConnection{
		ID:                id,
		SourceID:          req.SourceID,
		DestinationID:     req.DestinationID,
		ExternalID:        req.ExternalID,
		Enabled:           lo.FromPtr(req.Enabled),
		Schedule:          req.Schedule,
		SyncSettings:      req.SyncSettings,
		SyncBehaviour:     lo.FromPtr(req.SyncBehaviour),
		Identifiers:       req.Identifiers,
		Mappings:          req.Mappings,
		Event:             req.Event,
		Constants:         req.Constants,
		CursorColumn:      req.CursorColumn,
		Object:            req.Object,
		DestinationConfig: req.DestinationConfig,
	}
}

func (m *MockConnectionClient) UpdateConnection(_ context.Context, id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	m.UpdateCalls = append(m.UpdateCalls, id)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, req)
	}
	return &retlClient.RETLConnection{ID: id}, nil
}

func (m *MockConnectionClient) DeleteConnection(_ context.Context, id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockConnectionClient) ListConnections(_ context.Context, req *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
	m.ListCalls = append(m.ListCalls, *req)
	if m.ListFunc != nil {
		return m.ListFunc(req)
	}
	return &retlClient.RETLConnectionsPage{}, nil
}

func (m *MockConnectionClient) GetConnection(_ context.Context, id string) (*retlClient.RETLConnection, error) {
	m.GetCalls = append(m.GetCalls, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return &retlClient.RETLConnection{ID: id}, nil
}

func (m *MockConnectionClient) SetConnectionExternalId(_ context.Context, req *retlClient.SetRETLConnectionExternalIDRequest) error {
	m.SetExternalIDCalls = append(m.SetExternalIDCalls, *req)
	if m.SetExternalIDFunc != nil {
		return m.SetExternalIDFunc(req)
	}
	return nil
}

func (m *MockConnectionClient) ListRetlSources(_ context.Context, _ ...retlClient.ListRetlSourcesOption) (*retlClient.RETLSources, error) {
	m.SourceListCalls++
	if m.EndpointsErr != nil {
		return nil, m.EndpointsErr
	}
	return &retlClient.RETLSources{Data: m.Sources}, nil
}

func (m *MockConnectionClient) GetDestinations(_ context.Context) ([]apiClient.Destination, error) {
	m.DestinationsCalls++
	if m.EndpointsErr != nil {
		return nil, m.EndpointsErr
	}
	return m.Destinations, nil
}

func (m *MockConnectionClient) StartSync(_ context.Context, connectionID string, syncType retlClient.SyncType) (*retlClient.StartSyncResponse, error) {
	m.StartSyncCalls = append(m.StartSyncCalls, connectionID)
	if m.SyncErr != nil {
		return nil, m.SyncErr
	}
	if len(m.Syncs) > 0 {
		return &retlClient.StartSyncResponse{SyncID: m.Syncs[0].ID}, nil
	}
	return &retlClient.StartSyncResponse{SyncID: "run-1"}, nil
}

func (m *MockConnectionClient) StopSync(_ context.Context, connectionID string) error {
	m.StopSyncCalls = append(m.StopSyncCalls, connectionID)
	return m.SyncErr
}

func (m *MockConnectionClient) ListSyncs(_ context.Context, _ string, req retlClient.ListSyncsRequest) (*retlClient.SyncsPage, error) {
	m.ListSyncsCalls = append(m.ListSyncsCalls, req)
	if m.SyncErr != nil {
		return nil, m.SyncErr
	}
	return &retlClient.SyncsPage{
		Syncs:  m.Syncs,
		Paging: apiClient.Paging{Total: len(m.Syncs)},
	}, nil
}

func (m *MockConnectionClient) GetSync(_ context.Context, _, syncID string) (*retlClient.Sync, error) {
	m.GetSyncCalls = append(m.GetSyncCalls, syncID)
	if m.SyncErr != nil {
		return nil, m.SyncErr
	}
	for _, s := range m.Syncs {
		if s.ID == syncID {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("sync %q not found", syncID)
}
