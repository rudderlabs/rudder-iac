package connection

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	bingads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bingads_offline_conversions"
	customerioaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio_audience"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// importDir is the provider's import directory; export writes its single spec
// underneath it.
const importDir = "retl"

// testRegistry holds the destination definitions the remote fixtures name:
// http accepts warehouse sources through the JSON mapper, bingads also supports
// the visual mapper (object mapping), customerio_audience drives its own
// destination-specific rETL flow, and s3 takes no warehouse source at all.
func testRegistry(t *testing.T) *definitions.Registry {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))
	require.NoError(t, registry.Register(bingads.NewDefinition()))
	require.NoError(t, registry.Register(customerioaudience.NewDefinition()))
	require.NoError(t, registry.Register(s3.NewDefinition()))
	return registry
}

// remoteSources and remoteDestinations are the two endpoint catalogs the
// handler reads per operation. Between them they cover every eligibility
// outcome the handler has to tell apart.
func remoteSources() []retlClient.RETLSource {
	return []retlClient.RETLSource{
		{ID: "src-1", Name: "Users", SourceType: retlClient.ModelSourceType, WorkspaceID: "ws-1", ExternalID: "users"},
		{ID: "src-table", Name: "Tables", SourceType: retlClient.TableSourceType, WorkspaceID: "ws-1"},
		{ID: "src-no-workspace", Name: "Orphan", SourceType: retlClient.ModelSourceType},
	}
}

func remoteDestinations() []apiClient.Destination {
	return []apiClient.Destination{
		{ID: "dst-1", Name: "Webhook", Type: "HTTP", Version: 1, ExternalID: "webhook"},
		{ID: "dst-object", Name: "Bing Ads", Type: "BINGADS_OFFLINE_CONVERSIONS", Version: 1},
		{ID: "dst-specific", Name: "Customer IO", Type: "CUSTOMERIO_AUDIENCE", Version: 1},
		{ID: "dst-eventstream", Name: "S3 Bucket", Type: "S3", Version: 1},
		{ID: "dst-old", Name: "Legacy Webhook", Type: "HTTP", Version: 9},
	}
}

// remoteRow is one API connection row: the smallest representable JSON mapper
// config wired to the given endpoints.
func remoteRow(id, externalID, sourceID, destinationID string) retlClient.RETLConnection {
	conn := *representableConnection()
	conn.ID = id
	conn.ExternalID = externalID
	conn.SourceID = sourceID
	conn.DestinationID = destinationID
	conn.Enabled = true
	return conn
}

// eligibleRemote is the *RemoteConnection remoteConnection builds for a row
// wired to the shared endpoint fixtures — the payload both remote loaders
// store. Built through the same conversions the handler uses, so a fixture
// cannot drift from what the handler would produce.
func eligibleRemote(t *testing.T, conn retlClient.RETLConnection) *RemoteConnection {
	t.Helper()

	source, ok := lo.Find(remoteSources(), func(s retlClient.RETLSource) bool { return s.ID == conn.SourceID })
	require.True(t, ok, "source %q is not one of the shared fixtures", conn.SourceID)
	dst, ok := lo.Find(remoteDestinations(), func(d apiClient.Destination) bool { return d.ID == conn.DestinationID })
	require.True(t, ok, "destination %q is not one of the shared fixtures", conn.DestinationID)
	config, err := configFromRemote(&conn)
	require.NoError(t, err)
	// Resolved from the source's type rather than pinned to SourceKinds[0]:
	// with more than one kind registered, the index silently produced the
	// sql-model kind for a table-backed source.
	kind, ok := SourceKindBySourceType(source.SourceType)
	require.True(t, ok, "source type %q is not a registered source kind", source.SourceType)

	return &RemoteConnection{
		RETLConnection:        conn,
		Config:                config,
		WorkspaceID:           source.WorkspaceID,
		SourceKind:            kind,
		SourceName:            source.Name,
		SourceExternalID:      source.ExternalID,
		DestinationName:       dst.Name,
		DestinationExternalID: dst.ExternalID,
	}
}

// remoteClient is a client whose connections list serves the given pages in
// order and whose endpoint catalogs are the shared fixtures.
func remoteClient(pages ...[]retlClient.RETLConnection) *MockConnectionClient {
	return &MockConnectionClient{
		Sources:      remoteSources(),
		Destinations: remoteDestinations(),
		ListFunc:     pagedList(pages...),
	}
}

// pagedList serves one page per call, keyed on the requested page number. Every
// page but the last carries the internal gateway URL the CLI must never follow,
// and every page reports the row count the API reports: total.
func pagedList(pages ...[]retlClient.RETLConnection) func(*retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
	total := 0
	for _, rows := range pages {
		total += len(rows)
	}
	return func(req *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
		if req.Page < 1 || req.Page > len(pages) {
			return nil, fmt.Errorf("page %d requested but only %d pages exist", req.Page, len(pages))
		}
		page := &retlClient.RETLConnectionsPage{
			Data:   pages[req.Page-1],
			Paging: apiClient.Paging{Total: total},
		}
		if req.Page < len(pages) {
			page.Paging.Next = "/apigateway/v1/retl-connections?page=" + strconv.Itoa(req.Page+1)
		}
		return page, nil
	}
}

func remoteHandler(client *MockConnectionClient, t *testing.T) *Handler {
	t.Helper()
	return NewHandler(client, importDir, testRegistry(t))
}

func TestListPagination(t *testing.T) {
	t.Parallel()

	t.Run("walks the pages by number while next is set", func(t *testing.T) {
		t.Parallel()

		mock := remoteClient(
			[]retlClient.RETLConnection{remoteRow("conn-1", "", "src-1", "dst-1")},
			// A short page, then an empty one: neither ends a list whose next
			// is still set.
			nil,
			[]retlClient.RETLConnection{remoteRow("conn-2", "", "src-1", "dst-1")},
		)

		rows, err := remoteHandler(mock, t).List(t.Context(), nil)
		require.NoError(t, err)

		assert.Equal(t, []retlClient.ListRETLConnectionsRequest{
			{Page: 1, PageSize: 100},
			{Page: 2, PageSize: 100},
			{Page: 3, PageSize: 100},
		}, mock.ListCalls)
		assert.Len(t, rows, 2)
	})

	t.Run("a page without a next ends the list", func(t *testing.T) {
		t.Parallel()

		full := make([]retlClient.RETLConnection, listPageSize)
		for i := range full {
			full[i] = remoteRow(fmt.Sprintf("conn-%d", i), "", "src-1", "dst-1")
		}
		mock := remoteClient(full)

		rows, err := remoteHandler(mock, t).List(t.Context(), nil)
		require.NoError(t, err)

		assert.Len(t, mock.ListCalls, 1)
		assert.Len(t, rows, listPageSize)
	})

	t.Run("a later page failure fails the whole list", func(t *testing.T) {
		t.Parallel()

		mock := remoteClient([]retlClient.RETLConnection{remoteRow("conn-1", "", "src-1", "dst-1")}, nil)
		pages := mock.ListFunc
		mock.ListFunc = func(req *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
			if req.Page == 2 {
				return nil, errors.New("gateway timeout")
			}
			return pages(req)
		}

		rows, err := remoteHandler(mock, t).List(t.Context(), nil)

		assert.EqualError(t, err, "listing rETL connections (page 2): gateway timeout")
		assert.Nil(t, rows, "a partial list must never be reported as a successful one")
	})

	// next alone ends the walk: paging.total is advisory and the API discounts
	// it by the rows it skips on the page in hand, so a last page whose total
	// looks larger than the rows collected is still the end of the list.
	t.Run("an empty next ends the walk whatever total reports", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			ListFunc: func(_ *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
				return &retlClient.RETLConnectionsPage{
					Data:   []retlClient.RETLConnection{remoteRow("conn-1", "", "src-1", "dst-1")},
					Paging: apiClient.Paging{Total: 250},
				}, nil
			},
		}

		rows, err := remoteHandler(mock, t).List(t.Context(), nil)
		require.NoError(t, err)

		assert.Len(t, mock.ListCalls, 1)
		assert.Len(t, rows, 1)
	})

	t.Run("a nil page is an error, not an empty list", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			ListFunc: func(_ *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
				return nil, nil
			},
		}

		_, err := remoteHandler(mock, t).List(t.Context(), nil)

		assert.EqualError(t, err, "listing rETL connections (page 1): empty response")
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	managed := remoteRow("conn-1", "users-to-webhook", "src-1", "dst-1")
	unmanaged := remoteRow("conn-2", "", "src-1", "dst-object")
	unmanaged.Enabled = false
	mock := remoteClient([]retlClient.RETLConnection{managed, unmanaged})

	rows, err := remoteHandler(mock, t).List(t.Context(), lo.ToPtr(true))
	require.NoError(t, err)

	require.Len(t, mock.ListCalls, 1)
	assert.Equal(t, lo.ToPtr(true), mock.ListCalls[0].HasExternalID)
	assert.Equal(t, []resources.ResourceData{
		{
			IDKey:            "conn-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-1",
			EnabledKey:       true,
			ExternalIDKey:    "users-to-webhook",
		},
		{
			IDKey:            "conn-2",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-object",
			EnabledKey:       false,
			ExternalIDKey:    "",
		},
	}, rows)
}

func TestLoadResourcesFromRemote(t *testing.T) {
	t.Parallel()

	t.Run("keeps the managed rows the spec contract can express", func(t *testing.T) {
		t.Parallel()

		supported := remoteRow("conn-1", "users-to-webhook", "src-1", "dst-1")
		objectMapping := remoteRow("conn-object", "users-to-bingads", "src-1", "dst-object")
		objectMapping.Object = "Contact"

		unrepresentable := remoteRow("conn-config", "users-to-webhook-2", "src-1", "dst-1")
		unrepresentable.DestinationConfig = []byte(`{"listId":"42"}`)

		// A table-backed source is expressible now that retl-source-table is a
		// registered source kind. Before that it was dropped here alongside the
		// genuinely unrepresentable rows, which is the behaviour this PR changes.
		tableSource := remoteRow("conn-table-source", "b", "src-table", "dst-1")

		mock := remoteClient([]retlClient.RETLConnection{
			supported,
			objectMapping,
			unrepresentable,
			remoteRow("conn-missing-source", "a", "src-gone", "dst-1"),
			tableSource,
			remoteRow("conn-missing-destination", "c", "src-1", "dst-gone"),
			remoteRow("conn-unregistered-version", "d", "src-1", "dst-old"),
			remoteRow("conn-not-warehouse", "e", "src-1", "dst-eventstream"),
			remoteRow("conn-destination-specific", "f", "src-1", "dst-specific"),
			remoteRow("conn-no-workspace", "g", "src-no-workspace", "dst-1"),
		})

		collection, err := remoteHandler(mock, t).LoadResourcesFromRemote(t.Context())
		require.NoError(t, err)

		require.Len(t, mock.ListCalls, 1)
		assert.Equal(t, lo.ToPtr(true), mock.ListCalls[0].HasExternalID)
		assert.Equal(t, map[string]*resources.RemoteResource{
			"conn-1":            {ID: "conn-1", ExternalID: "users-to-webhook", Data: eligibleRemote(t, supported)},
			"conn-object":       {ID: "conn-object", ExternalID: "users-to-bingads", Data: eligibleRemote(t, objectMapping)},
			"conn-table-source": {ID: "conn-table-source", ExternalID: "b", Data: eligibleRemote(t, tableSource)},
		}, collection.GetAll(ResourceType))
	})

	t.Run("an empty workspace never reaches for the endpoint catalogs", func(t *testing.T) {
		t.Parallel()

		mock := remoteClient(nil)
		mock.EndpointsErr = errors.New("must not be called")

		collection, err := remoteHandler(mock, t).LoadResourcesFromRemote(t.Context())

		require.NoError(t, err)
		assert.Empty(t, collection.GetAll(ResourceType))
	})

	t.Run("surfaces endpoint lookup errors", func(t *testing.T) {
		t.Parallel()

		mock := remoteClient([]retlClient.RETLConnection{remoteRow("conn-1", "users-to-webhook", "src-1", "dst-1")})
		mock.EndpointsErr = errors.New("forbidden")

		_, err := remoteHandler(mock, t).LoadResourcesFromRemote(t.Context())

		assert.EqualError(t, err, "listing rETL sources: forbidden")
	})
}

// managedCollection is what the syncer hands MapRemoteToState: this handler's
// rows merged with the endpoint providers' managed resources. The payload is
// the *RemoteConnection LoadResourcesFromRemote stores, config included.
func managedCollection(t *testing.T, conns ...retlClient.RETLConnection) *resources.RemoteResources {
	t.Helper()

	collection := resources.NewRemoteResources()
	connectionMap := make(map[string]*resources.RemoteResource, len(conns))
	for _, conn := range conns {
		// Only the row and its rebuilt config matter here: MapRemoteToState
		// resolves the endpoints out of the collection, not the catalogs, which
		// is what lets these fixtures name endpoints the catalogs do not hold.
		config, err := configFromRemote(&conn)
		require.NoError(t, err)
		connectionMap[conn.ID] = &resources.RemoteResource{
			ID:         conn.ID,
			ExternalID: conn.ExternalID,
			Data:       &RemoteConnection{RETLConnection: conn, Config: config},
		}
	}
	collection.Set(ResourceType, connectionMap)
	collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
		"src-1":         {ID: "src-1", ExternalID: "users"},
		"src-unmanaged": {ID: "src-unmanaged"},
	})
	collection.Set(destination.DestinationResourceType, map[string]*resources.RemoteResource{
		"dst-1": {ID: "dst-1", ExternalID: "webhook"},
	})
	return collection
}

func TestMapRemoteToState(t *testing.T) {
	t.Parallel()

	t.Run("state input mirrors the spec side exactly", func(t *testing.T) {
		t.Parallel()

		conn := remoteRow("conn-1", "users-to-webhook", "src-1", "dst-1")
		s, err := remoteHandler(remoteClient(), t).MapRemoteToState(managedCollection(t, conn))
		require.NoError(t, err)

		// The spec side's canonical config for the same connection: local and
		// remote have to land on the identical map or every apply re-diffs.
		spec := newHandler()
		require.NoError(t, spec.LoadSpec("", connectionsSpec(specBody(specEntry(nil)))))
		local := spec.resources["users-to-webhook"]

		assert.Equal(t, map[string]*state.ResourceState{
			"retl-connection:users-to-webhook": {
				ID:   "users-to-webhook",
				Type: ResourceType,
				Input: map[string]any{
					SourceKey:      &resources.PropertyRef{URN: "retl-source-sql-model:users", Property: "id"},
					DestinationKey: &resources.PropertyRef{URN: "destination:webhook", Property: "id"},
					EnabledKey:     true,
					ConfigKey:      local.Config,
				},
				Output: map[string]any{
					IDKey:            "conn-1",
					SourceIDKey:      "src-1",
					DestinationIDKey: "dst-1",
				},
			},
		}, s.Resources)
		assert.Equal(t, local.Source, s.Resources["retl-connection:users-to-webhook"].Input[SourceKey])
	})

	t.Run("skips rows whose endpoints are not managed", func(t *testing.T) {
		t.Parallel()

		s, err := remoteHandler(remoteClient(), t).MapRemoteToState(managedCollection(t,
			remoteRow("conn-1", "users-to-webhook", "src-1", "dst-1"),
			remoteRow("conn-2", "unknown-source", "src-gone", "dst-1"),
			remoteRow("conn-3", "unmanaged-source", "src-unmanaged", "dst-1"),
			remoteRow("conn-4", "unknown-destination", "src-1", "dst-gone"),
		))
		require.NoError(t, err)

		require.Len(t, s.Resources, 1)
		assert.Contains(t, s.Resources, "retl-connection:users-to-webhook")
	})

	t.Run("errors on foreign data in the collection", func(t *testing.T) {
		t.Parallel()

		collection := resources.NewRemoteResources()
		collection.Set(ResourceType, map[string]*resources.RemoteResource{
			"x": {ID: "x", ExternalID: "x", Data: "not a connection"},
		})

		_, err := remoteHandler(remoteClient(), t).MapRemoteToState(collection)

		assert.ErrorContains(t, err, "unable to cast resource to rETL connection")
	})
}
