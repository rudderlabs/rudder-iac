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

// reservedMappingRow is a JSON mapper row whose user mappings aim at a target
// the backend reserves for identifiers. Re-applying the spec it would produce
// moves the entry into the identifiers, so the row would diff forever.
func reservedMappingRow() retlClient.RETLConnection {
	conn := remoteRow("conn-reserved-mapping", "", "src-1", "dst-1")
	conn.Mappings = append(conn.Mappings, retlClient.Mapping{From: "device", To: AnonymousIDTarget})
	return conn
}

// manyIdentifiersRow is an object mapping row carrying two identifiers; the
// flow stores a single one, so the second could never be recreated.
func manyIdentifiersRow() retlClient.RETLConnection {
	conn := remoteRow("conn-many-identifiers", "", "src-1", "dst-object")
	conn.Object = "Contact"
	conn.Identifiers = append(conn.Identifiers, retlClient.Mapping{From: "email", To: "Email"})
	return conn
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

	// The API drops next exactly when page*pageSize reaches total, so a last
	// page whose own total promises more rows contradicts itself: reporting the
	// rows collected so far as the whole list would silently truncate it.
	t.Run("a last page that still promises more rows is an error", func(t *testing.T) {
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

		assert.EqualError(t, err, "listing rETL connections: page 1 reported 250 connections in total but no further page")
		assert.Nil(t, rows)
	})

	t.Run("a next that never ends stops at the page bound", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			ListFunc: func(req *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
				return &retlClient.RETLConnectionsPage{
					Data:   []retlClient.RETLConnection{remoteRow("conn-"+strconv.Itoa(req.Page), "", "src-1", "dst-1")},
					Paging: apiClient.Paging{Next: "/apigateway/v1/retl-connections?page=" + strconv.Itoa(req.Page+1)},
				}, nil
			},
		}

		rows, err := remoteHandler(mock, t).List(t.Context(), nil)

		assert.EqualError(t, err, "listing rETL connections: the API kept reporting another page past page 1000")
		assert.Nil(t, rows, "a walk that never terminated must never be reported as a successful list")
		assert.Len(t, mock.ListCalls, maxListPages)
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

		mock := remoteClient([]retlClient.RETLConnection{
			supported,
			objectMapping,
			unrepresentable,
			remoteRow("conn-missing-source", "a", "src-gone", "dst-1"),
			remoteRow("conn-table-source", "b", "src-table", "dst-1"),
			remoteRow("conn-missing-destination", "c", "src-1", "dst-gone"),
			remoteRow("conn-unregistered-version", "d", "src-1", "dst-old"),
			remoteRow("conn-not-warehouse", "e", "src-1", "dst-eventstream"),
			remoteRow("conn-destination-specific", "f", "src-1", "dst-specific"),
			remoteRow("conn-no-workspace", "g", "src-no-workspace", "dst-1"),
			reservedMappingRow(),
			manyIdentifiersRow(),
		})

		collection, err := remoteHandler(mock, t).LoadResourcesFromRemote(t.Context())
		require.NoError(t, err)

		require.Len(t, mock.ListCalls, 1)
		assert.Equal(t, lo.ToPtr(true), mock.ListCalls[0].HasExternalID)
		assert.Equal(t, map[string]*resources.RemoteResource{
			"conn-1":      {ID: "conn-1", ExternalID: "users-to-webhook", Data: supported},
			"conn-object": {ID: "conn-object", ExternalID: "users-to-bingads", Data: objectMapping},
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
// rows merged with the endpoint providers' managed resources.
func managedCollection(conns ...retlClient.RETLConnection) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	connectionMap := make(map[string]*resources.RemoteResource, len(conns))
	for _, conn := range conns {
		connectionMap[conn.ID] = &resources.RemoteResource{ID: conn.ID, ExternalID: conn.ExternalID, Data: conn}
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
		s, err := remoteHandler(remoteClient(), t).MapRemoteToState(managedCollection(conn))
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

		s, err := remoteHandler(remoteClient(), t).MapRemoteToState(managedCollection(
			remoteRow("conn-1", "users-to-webhook", "src-1", "dst-1"),
			remoteRow("conn-2", "unknown-source", "src-gone", "dst-1"),
			remoteRow("conn-3", "unmanaged-source", "src-unmanaged", "dst-1"),
			remoteRow("conn-4", "unknown-destination", "src-1", "dst-gone"),
		))
		require.NoError(t, err)

		require.Len(t, s.Resources, 1)
		assert.Contains(t, s.Resources, "retl-connection:users-to-webhook")
	})

	t.Run("a config the spec cannot express is an error, not a silent drop", func(t *testing.T) {
		t.Parallel()

		conn := remoteRow("conn-1", "users-to-webhook", "src-1", "dst-1")
		conn.Identifiers = nil

		_, err := remoteHandler(remoteClient(), t).MapRemoteToState(managedCollection(conn))

		assert.ErrorContains(t, err, `reading remote connection "users-to-webhook"`)
		assert.ErrorIs(t, err, ErrUnrepresentableConfig)
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
