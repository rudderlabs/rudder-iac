package table_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// fakeStore is an in-memory RETL source API. It mirrors the server behaviour
// the handler depends on: create accepts an external id, update cannot change
// the source definition, and list honours the sourceType and hasExternalId
// filters. Calls are recorded so tests can assert what reached the API.
type fakeStore struct {
	retlClient.RETLStore
	sources map[string]*retlClient.RETLSource
	nextID  int
	calls   []string
	failOn  string
}

func newFakeStore(existing ...retlClient.RETLSource) *fakeStore {
	f := &fakeStore{sources: make(map[string]*retlClient.RETLSource)}
	for _, s := range existing {
		f.sources[s.ID] = &s
	}
	return f
}

func (f *fakeStore) record(call string) error {
	f.calls = append(f.calls, call)
	if f.failOn == call {
		return fmt.Errorf("api error on %s", call)
	}
	return nil
}

func (f *fakeStore) CreateRetlSource(_ context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
	if err := f.record("create"); err != nil {
		return nil, err
	}
	f.nextID++
	s := &retlClient.RETLSource{
		ID:                   fmt.Sprintf("src-%d", f.nextID),
		Name:                 req.Name,
		Config:               req.Config,
		IsEnabled:            req.Enabled,
		SourceType:           req.SourceType,
		SourceDefinitionName: req.SourceDefinitionName,
		AccountID:            req.AccountID,
		WorkspaceID:          "ws-1",
		ExternalID:           req.ExternalID,
	}
	f.sources[s.ID] = s
	copied := *s
	return &copied, nil
}

func (f *fakeStore) UpdateRetlSource(_ context.Context, id string, req *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
	if err := f.record("update:" + id); err != nil {
		return nil, err
	}
	s, ok := f.sources[id]
	if !ok {
		return nil, fmt.Errorf("source %s not found", id)
	}
	s.Name = req.Name
	s.Config = req.Config
	s.IsEnabled = req.IsEnabled
	s.AccountID = req.AccountID
	copied := *s
	return &copied, nil
}

func (f *fakeStore) DeleteRetlSource(_ context.Context, id string) error {
	if err := f.record("delete:" + id); err != nil {
		return err
	}
	delete(f.sources, id)
	return nil
}

func (f *fakeStore) GetRetlSource(_ context.Context, id string) (*retlClient.RETLSource, error) {
	if err := f.record("get:" + id); err != nil {
		return nil, err
	}
	s, ok := f.sources[id]
	if !ok {
		return nil, fmt.Errorf("source %s not found", id)
	}
	copied := *s
	return &copied, nil
}

func (f *fakeStore) SetExternalId(_ context.Context, id string, externalID string) error {
	if err := f.record("setExternalId:" + id); err != nil {
		return err
	}
	f.sources[id].ExternalID = externalID
	return nil
}

func (f *fakeStore) ListRetlSources(_ context.Context, opts ...retlClient.ListRetlSourcesOption) (*retlClient.RETLSources, error) {
	var o retlClient.ListRetlSourcesOptions
	for _, opt := range opts {
		opt(&o)
	}
	call := "list:" + o.SourceType
	if o.HasExternalId != nil {
		call += fmt.Sprintf(":%t", *o.HasExternalId)
	}
	if err := f.record(call); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(f.sources))
	for id := range f.sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	result := &retlClient.RETLSources{}
	for _, id := range ids {
		s := f.sources[id]
		if o.SourceType != "" && string(s.SourceType) != o.SourceType {
			continue
		}
		if o.HasExternalId != nil && (s.ExternalID != "") != *o.HasExternalId {
			continue
		}
		result.Data = append(result.Data, *s)
	}
	return result, nil
}

func warehouseSpec() *specs.Spec {
	return &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    table.ResourceKind,
		Spec: map[string]any{
			"id":                "users-table",
			"display_name":      "Users",
			"account_id":        "acc-123",
			"source_definition": "postgres",
			"primary_key":       "id",
			"schema":            "public",
			"table":             "users",
		},
	}
}

func s3Spec() *specs.Spec {
	return &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    table.ResourceKind,
		Spec: map[string]any{
			"id":                "events-bucket",
			"display_name":      "Events",
			"account_id":        "acc-s3",
			"source_definition": "s3",
			"bucket_name":       "events",
			"object_prefix":     "daily/",
		},
	}
}

func withField(s *specs.Spec, key string, value any) *specs.Spec {
	s.Spec[key] = value
	return s
}

func withoutField(s *specs.Spec, key string) *specs.Spec {
	delete(s.Spec, key)
	return s
}

// loadResource loads spec into a fresh handler and returns its single resource.
func loadResource(t *testing.T, store retlClient.RETLStore, spec *specs.Spec) (*table.Handler, *resources.Resource) {
	t.Helper()
	h := table.NewHandler(store, "retl")
	require.NoError(t, h.LoadSpec("spec.yaml", spec))
	got, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, got, 1)
	return h, got[0]
}

func TestLoadSpec(t *testing.T) {
	t.Parallel()

	t.Run("warehouse spec becomes graph data with enabled defaulting to true", func(t *testing.T) {
		t.Parallel()
		_, r := loadResource(t, newFakeStore(), warehouseSpec())

		assert.Equal(t, "users-table", r.ID())
		assert.Equal(t, table.ResourceType, r.Type())
		assert.Equal(t, resources.ResourceData{
			sqlmodel.LocalIDKey:          "users-table",
			sqlmodel.DisplayNameKey:      "Users",
			sqlmodel.AccountIDKey:        "acc-123",
			sqlmodel.SourceDefinitionKey: "postgres",
			sqlmodel.PrimaryKeyKey:       "id",
			table.SchemaKey:              "public",
			table.TableKey:               "users",
			sqlmodel.EnabledKey:          true,
		}, r.Data())
	})

	t.Run("s3 spec needs no primary key", func(t *testing.T) {
		t.Parallel()
		_, r := loadResource(t, newFakeStore(), withField(s3Spec(), "enabled", false))

		assert.Equal(t, resources.ResourceData{
			sqlmodel.LocalIDKey:          "events-bucket",
			sqlmodel.DisplayNameKey:      "Events",
			sqlmodel.AccountIDKey:        "acc-s3",
			sqlmodel.SourceDefinitionKey: "s3",
			sqlmodel.PrimaryKeyKey:       "",
			table.BucketNameKey:          "events",
			table.ObjectPrefixKey:        "daily/",
			sqlmodel.EnabledKey:          false,
		}, r.Data())
	})

	// Field rules live in retl/table/spec-syntax-valid; LoadSpec only refuses
	// keys the spec has no field for.
	invalid := []struct {
		name    string
		spec    *specs.Spec
		wantErr string
	}{
		{
			// A table config carries no description, so it could never round-trip.
			name:    "description",
			spec:    withField(warehouseSpec(), "description", "Customer records"),
			wantErr: "decoding table source spec",
		},
		{
			name:    "nested config block",
			spec:    withField(warehouseSpec(), "config", map[string]any{"schema": "public"}),
			wantErr: "decoding table source spec",
		},
	}
	for _, tc := range invalid {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			t.Parallel()
			h := table.NewHandler(newFakeStore(), "retl")

			err := h.LoadSpec("spec.yaml", tc.spec)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			got, err := h.GetResources()
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	}

	t.Run("rejects a duplicate id", func(t *testing.T) {
		t.Parallel()
		h := table.NewHandler(newFakeStore(), "retl")
		require.NoError(t, h.LoadSpec("a.yaml", warehouseSpec()))

		err := h.LoadSpec("b.yaml", warehouseSpec())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "table source with id users-table already exists")
	})
}

func TestParseSpec(t *testing.T) {
	t.Parallel()

	t.Run("returns the spec URN without a legacy resource type", func(t *testing.T) {
		t.Parallel()
		h := table.NewHandler(newFakeStore(), "retl")

		parsed, err := h.ParseSpec("spec.yaml", warehouseSpec())

		require.NoError(t, err)
		// No legacy resource type, so metadata-syntax-valid rejects local_id
		// import entries for this v1-only kind.
		assert.Equal(t, &specs.ParsedSpec{
			URNs: []specs.URNEntry{{URN: "retl-source-table:users-table", JSONPointerPath: "/spec/id"}},
		}, parsed)
	})

	t.Run("errors without an id", func(t *testing.T) {
		t.Parallel()
		h := table.NewHandler(newFakeStore(), "retl")

		_, err := h.ParseSpec("spec.yaml", withoutField(warehouseSpec(), "id"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "id not found")
	})
}

func TestImportMetadata(t *testing.T) {
	t.Parallel()

	spec := warehouseSpec()
	spec.Metadata = map[string]any{
		"name": "users-table",
		"import": map[string]any{
			"workspaces": []any{map[string]any{
				"workspace_id": "ws-1",
				"resources": []any{
					map[string]any{"urn": "retl-source-table:users-table", "remote_id": "src-remote"},
				},
			}},
		},
	}

	t.Run("inline metadata.import attaches the remote id", func(t *testing.T) {
		t.Parallel()
		_, r := loadResource(t, newFakeStore(), spec)

		require.NotNil(t, r.ImportMetadata())
		assert.Equal(t, "src-remote", r.ImportMetadata().RemoteId)
		assert.Equal(t, "ws-1", r.ImportMetadata().WorkspaceId)
	})

	t.Run("manifest entries apply by URN and ignore legacy local_id", func(t *testing.T) {
		t.Parallel()
		h := table.NewHandler(newFakeStore(), "retl")
		require.NoError(t, h.LoadSpec("spec.yaml", warehouseSpec()))

		require.NoError(t, h.LoadImportMetadata(&specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{{
				WorkspaceID: "ws-1",
				Resources:   []specs.ImportIds{{LocalID: "users-table", RemoteID: "src-legacy"}},
			}},
		}))
		got, err := h.GetResources()
		require.NoError(t, err)
		assert.Nil(t, got[0].ImportMetadata())

		require.NoError(t, h.LoadImportMetadata(&specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{{
				WorkspaceID: "ws-1",
				Resources:   []specs.ImportIds{{URN: "retl-source-table:users-table", RemoteID: "src-manifest"}},
			}},
		}))
		got, err = h.GetResources()
		require.NoError(t, err)
		require.NotNil(t, got[0].ImportMetadata())
		assert.Equal(t, "src-manifest", got[0].ImportMetadata().RemoteId)
	})
}

func TestLifecycle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		spec       func() *specs.Spec
		wantConfig retlClient.RETLConfig
		change     func(*specs.Spec) *specs.Spec
		wantUpdate retlClient.RETLConfig
	}{
		{
			name:       "warehouse",
			spec:       warehouseSpec,
			wantConfig: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
			change: func(s *specs.Spec) *specs.Spec {
				return withField(withField(s, "schema", "analytics"), "primary_key", "user_id")
			},
			wantUpdate: retlClient.RETLTableConfig{PrimaryKey: "user_id", Schema: "analytics", Table: "users"},
		},
		{
			name:       "s3",
			spec:       s3Spec,
			wantConfig: retlClient.RETLS3TableConfig{BucketName: "events", ObjectPrefix: "daily/"},
			change: func(s *specs.Spec) *specs.Spec {
				return withField(withField(s, "bucket_name", "events-v2"), "object_prefix", "hourly/")
			},
			wantUpdate: retlClient.RETLS3TableConfig{BucketName: "events-v2", ObjectPrefix: "hourly/"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store := newFakeStore()
			h, r := loadResource(t, store, tc.spec())

			// Create sends a table source carrying the local id as external id.
			output, err := h.Create(ctx, r.ID(), r.Data())
			require.NoError(t, err)
			sourceID := (*output)[sqlmodel.IDKey].(string)
			created := store.sources[sourceID]
			assert.Equal(t, retlClient.TableSourceType, created.SourceType)
			assert.Equal(t, r.ID(), created.ExternalID)
			assert.Equal(t, r.Data()[sqlmodel.SourceDefinitionKey], created.SourceDefinitionName)
			assert.Equal(t, tc.wantConfig, created.Config)

			// Remote state rebuilds exactly the local graph data, so a plan
			// right after apply shows no change.
			remote, err := h.LoadResourcesFromRemote(ctx)
			require.NoError(t, err)
			st, err := h.MapRemoteToState(remote)
			require.NoError(t, err)
			rs := st.GetResource(r.URN())
			require.NotNil(t, rs)
			assert.Equal(t, map[string]any(r.Data()), rs.Input)
			assert.Equal(t, sourceID, rs.Output[sqlmodel.IDKey])

			// Update sends every mutable field.
			_, changed := loadResource(t, store, tc.change(tc.spec()))
			_, err = h.Update(ctx, r.ID(), changed.Data(), rs.Data())
			require.NoError(t, err)
			assert.Equal(t, tc.wantUpdate, store.sources[sourceID].Config)

			require.NoError(t, h.Delete(ctx, r.ID(), rs.Data()))
			assert.NotContains(t, store.sources, sourceID)
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("rejects a source_definition change without calling the API", func(t *testing.T) {
		t.Parallel()
		store := newFakeStore()
		h, r := loadResource(t, store, withField(warehouseSpec(), "source_definition", "snowflake"))
		state := resources.ResourceData{
			sqlmodel.IDKey:               "src-1",
			sqlmodel.SourceDefinitionKey: "postgres",
		}

		_, err := h.Update(context.Background(), r.ID(), r.Data(), state)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `source_definition cannot be changed from "postgres" to "snowflake"`)
		assert.Empty(t, store.calls)
	})

	t.Run("errors when state has no remote id", func(t *testing.T) {
		t.Parallel()
		h, r := loadResource(t, newFakeStore(), warehouseSpec())

		_, err := h.Update(context.Background(), r.ID(), r.Data(), resources.ResourceData{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing id in resource state")
	})

	t.Run("wraps API errors", func(t *testing.T) {
		t.Parallel()
		store := newFakeStore(retlClient.RETLSource{ID: "src-1", SourceType: retlClient.TableSourceType})
		store.failOn = "update:src-1"
		h, r := loadResource(t, store, warehouseSpec())
		state := resources.ResourceData{sqlmodel.IDKey: "src-1", sqlmodel.SourceDefinitionKey: "postgres"}

		_, err := h.Update(context.Background(), r.ID(), r.Data(), state)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "updating RETL source: api error on update:src-1")
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("errors when state has no remote id", func(t *testing.T) {
		t.Parallel()
		store := newFakeStore()
		h := table.NewHandler(store, "retl")

		err := h.Delete(context.Background(), "users-table", resources.ResourceData{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing id in resource state")
		assert.Empty(t, store.calls)
	})

	t.Run("wraps API errors", func(t *testing.T) {
		t.Parallel()
		store := newFakeStore()
		store.failOn = "delete:src-1"
		h := table.NewHandler(store, "retl")

		err := h.Delete(context.Background(), "users-table", resources.ResourceData{sqlmodel.IDKey: "src-1"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deleting RETL source")
	})
}

func TestImport(t *testing.T) {
	t.Parallel()

	remoteUsers := func() retlClient.RETLSource {
		return retlClient.RETLSource{
			ID:                   "src-remote",
			Name:                 "Users",
			SourceType:           retlClient.TableSourceType,
			SourceDefinitionName: "postgres",
			AccountID:            "acc-123",
			IsEnabled:            true,
			Config:               retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
		}
	}

	t.Run("claims an in-sync source without updating it", func(t *testing.T) {
		t.Parallel()
		store := newFakeStore(remoteUsers())
		h, r := loadResource(t, store, warehouseSpec())

		output, err := h.Import(context.Background(), r.ID(), r.Data(), "src-remote")

		require.NoError(t, err)
		assert.Equal(t, []string{"get:src-remote", "setExternalId:src-remote"}, store.calls)
		assert.Equal(t, "users-table", store.sources["src-remote"].ExternalID)
		assert.Equal(t, "src-remote", (*output)[sqlmodel.IDKey])
	})

	t.Run("updates a diverged source to match the spec", func(t *testing.T) {
		t.Parallel()
		diverged := remoteUsers()
		diverged.Config = retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "staging", Table: "users"}
		store := newFakeStore(diverged)
		h, r := loadResource(t, store, warehouseSpec())

		_, err := h.Import(context.Background(), r.ID(), r.Data(), "src-remote")

		require.NoError(t, err)
		assert.Equal(t, []string{"get:src-remote", "setExternalId:src-remote", "update:src-remote"}, store.calls)
		assert.Equal(t, retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"}, store.sources["src-remote"].Config)
	})

	t.Run("refuses a source_definition mismatch before claiming the source", func(t *testing.T) {
		t.Parallel()
		store := newFakeStore(remoteUsers())
		h, r := loadResource(t, store, withField(warehouseSpec(), "source_definition", "snowflake"))

		_, err := h.Import(context.Background(), r.ID(), r.Data(), "src-remote")

		require.Error(t, err)
		assert.Contains(t, err.Error(), `source_definition is "postgres" remotely and "snowflake" locally`)
		assert.Equal(t, []string{"get:src-remote"}, store.calls)
		assert.Empty(t, store.sources["src-remote"].ExternalID)
	})

	t.Run("refuses a source that is not a table source", func(t *testing.T) {
		t.Parallel()
		model := remoteUsers()
		model.SourceType = retlClient.ModelSourceType
		model.Config = retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: "SELECT 1"}
		store := newFakeStore(model)
		h, r := loadResource(t, store, warehouseSpec())

		_, err := h.Import(context.Background(), r.ID(), r.Data(), "src-remote")

		require.Error(t, err)
		assert.Contains(t, err.Error(), `source type is "model", not "table"`)
		assert.Equal(t, []string{"get:src-remote"}, store.calls)
	})
}

func TestRemoteReadsOnlyTableSources(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		retlClient.RETLSource{
			ID: "src-managed", Name: "Users", ExternalID: "users-table",
			SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres",
			Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
		},
		retlClient.RETLSource{
			ID: "src-unmanaged", Name: "Orders",
			SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres",
			Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "orders"},
		},
		retlClient.RETLSource{
			ID: "src-model", Name: "Model", ExternalID: "a-model",
			SourceType: retlClient.ModelSourceType, SourceDefinitionName: "postgres",
			Config: retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: "SELECT 1"},
		},
	)
	h := table.NewHandler(store, "retl")

	remote, err := h.LoadResourcesFromRemote(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"list:table:true"}, store.calls)
	got := remote.GetAll(table.ResourceType)
	require.Len(t, got, 1)
	assert.Equal(t, "users-table", got["src-managed"].ExternalID)
}

func TestImportWorkspace(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		retlClient.RETLSource{
			ID: "src-1", Name: "Users", WorkspaceID: "ws-1", AccountID: "acc-123", IsEnabled: true,
			SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres",
			Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
		},
		retlClient.RETLSource{
			ID: "src-2", Name: "Raw Events", WorkspaceID: "ws-1", AccountID: "acc-s3",
			SourceType: retlClient.TableSourceType, SourceDefinitionName: "s3",
			Config: retlClient.RETLS3TableConfig{BucketName: "events"},
		},
		retlClient.RETLSource{
			ID: "src-managed", Name: "Managed", ExternalID: "managed", WorkspaceID: "ws-1",
			SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres",
			Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "managed"},
		},
	)
	h := table.NewHandler(store, "retl")

	importable, err := h.LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.StrategyKebabCase))
	require.NoError(t, err)
	assert.Equal(t, []string{"list:table:false"}, store.calls)
	require.Len(t, importable.GetAll(table.ResourceType), 2)
	users, ok := importable.GetByID(table.ResourceType, "src-1")
	require.True(t, ok)
	assert.Equal(t, "users", users.ExternalID)
	assert.Equal(t, "#retl-source-table:users", users.Reference)

	entities, entries, err := h.FormatForExport(importable, nil, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws-1", URN: "retl-source-table:users", RemoteID: "src-1"},
		{WorkspaceID: "ws-1", URN: "retl-source-table:raw-events", RemoteID: "src-2"},
	}, entries)
	require.Len(t, entities, 2)

	byPath := make(map[string]*specs.Spec, len(entities))
	for _, e := range entities {
		spec, ok := e.Content.(*specs.Spec)
		require.True(t, ok)
		byPath[e.RelativePath] = spec
	}
	usersSpec := byPath["retl/tables/users.yaml"]
	require.NotNil(t, usersSpec)
	assert.Equal(t, specs.SpecVersionV1, usersSpec.Version)
	assert.Equal(t, table.ResourceKind, usersSpec.Kind)
	assert.Equal(t, map[string]any{
		"id":                "users",
		"display_name":      "Users",
		"account_id":        "acc-123",
		"source_definition": "postgres",
		"primary_key":       "id",
		"schema":            "public",
		"table":             "users",
		"enabled":           true,
	}, usersSpec.Spec)
	eventsSpec := byPath["retl/tables/raw-events.yaml"]
	require.NotNil(t, eventsSpec)
	assert.Equal(t, map[string]any{
		"id":                "raw-events",
		"display_name":      "Raw Events",
		"account_id":        "acc-s3",
		"source_definition": "s3",
		"bucket_name":       "events",
		"enabled":           false,
	}, eventsSpec.Spec)

	// Every exported spec loads back, and carries the import link.
	for path, spec := range byPath {
		_, r := loadResource(t, newFakeStore(), spec)
		require.NotNil(t, r.ImportMetadata(), path)
		assert.Equal(t, "ws-1", r.ImportMetadata().WorkspaceId, path)
	}
}

func TestFormatForExportSkipsMatched(t *testing.T) {
	t.Parallel()
	h := table.NewHandler(newFakeStore(), "retl")

	collection := resources.NewRemoteResources()
	collection.Set(table.ResourceType, map[string]*resources.RemoteResource{
		"src-1": {
			ID:         "src-1",
			ExternalID: "users-table",
			Data: &retlClient.RETLSource{
				ID: "src-1", Name: "Users", WorkspaceID: "ws-1",
				SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres",
				Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
			},
			MatchedWith: resources.NewResource("users-table", table.ResourceType, resources.ResourceData{}, []string{}),
		},
	})

	entities, entries, err := h.FormatForExport(collection, nil, nil)

	require.NoError(t, err)
	assert.Empty(t, entities, "a matched source adopts the existing local spec")
	assert.Equal(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws-1", URN: "retl-source-table:users-table", RemoteID: "src-1"},
	}, entries)
}

func TestList(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		retlClient.RETLSource{
			ID: "src-1", Name: "Users", AccountID: "acc-123",
			SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres",
			Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
		},
		retlClient.RETLSource{
			ID: "src-2", Name: "Raw Events", AccountID: "acc-s3",
			SourceType: retlClient.TableSourceType, SourceDefinitionName: "s3",
			Config: retlClient.RETLS3TableConfig{BucketName: "events", ObjectPrefix: "daily/"},
		},
	)
	h := table.NewHandler(store, "retl")

	got, err := h.List(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"list:table"}, store.calls)
	require.Len(t, got, 2)
	assert.Equal(t, map[string]any{"primary_key": "id", "schema": "public", "table": "users"}, got[0]["config"])
	assert.Equal(t, map[string]any{"bucket_name": "events", "object_prefix": "daily/"}, got[1]["config"])
}

func TestUnsupportedOperations(t *testing.T) {
	t.Parallel()
	h := table.NewHandler(newFakeStore(), "retl")

	_, err := h.Preview(context.Background(), "users-table", resources.ResourceData{}, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "preview is not supported for retl-source-table")

	_, err = h.FetchImportData(context.Background(), specs.ImportIds{LocalID: "users", RemoteID: "src-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "single-source import is not supported for retl-source-table")
}
