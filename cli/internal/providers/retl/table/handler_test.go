package table_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// mockStore embeds RETLStore so unimplemented methods are nil-panics rather
// than compile errors; each test overrides only what it exercises.
type mockStore struct {
	retlClient.RETLStore
	createFunc func(ctx context.Context, s *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error)
	updateFunc func(ctx context.Context, id string, s *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error)
	deleteFunc func(ctx context.Context, id string) error
}

func (m *mockStore) CreateRetlSource(ctx context.Context, s *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
	return m.createFunc(ctx, s)
}

func (m *mockStore) UpdateRetlSource(ctx context.Context, id string, s *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
	return m.updateFunc(ctx, id, s)
}

func (m *mockStore) DeleteRetlSource(ctx context.Context, id string) error {
	return m.deleteFunc(ctx, id)
}

func validSpec() *specs.Spec {
	return &specs.Spec{
		Version: "rudder/v1",
		Kind:    table.ResourceKind,
		Spec: map[string]any{
			"id":                "users-table",
			"display_name":      "Users",
			"account_id":        "acc-123",
			"source_definition": "snowflake",
			"primary_key":       "id",
			"schema":            "public",
			"table":             "users",
		},
	}
}

func TestLoadSpec(t *testing.T) {
	t.Run("loads a valid spec and defaults enabled to true", func(t *testing.T) {
		h := table.NewHandler(&mockStore{}, t.TempDir())
		require.NoError(t, h.LoadSpec("spec.yaml", validSpec()))

		got, err := h.GetResources()
		require.NoError(t, err)
		require.Len(t, got, 1)

		data := got[0].Data()
		assert.Equal(t, "Users", data[table.DisplayNameKey])
		assert.NotContains(t, data, "description",
			"RETLTableConfig has no Description field, so a description could never round-trip")
		assert.Equal(t, "public", data[table.SchemaKey])
		assert.Equal(t, "users", data[table.TableKey])
		assert.Equal(t, "id", data[table.PrimaryKeyKey])
		assert.Equal(t, true, data[table.EnabledKey], "enabled should default to true")
	})

	t.Run("rejects an unknown field", func(t *testing.T) {
		h := table.NewHandler(&mockStore{}, t.TempDir())
		s := validSpec()
		s.Spec["bucket_name"] = "not-valid-here"

		err := h.LoadSpec("spec.yaml", s)
		require.Error(t, err, "s3 keys must not be silently accepted on a warehouse table spec")
	})

	t.Run("rejects an invalid source_definition", func(t *testing.T) {
		h := table.NewHandler(&mockStore{}, t.TempDir())
		s := validSpec()
		s.Spec["source_definition"] = "s3"

		err := h.LoadSpec("spec.yaml", s)
		require.Error(t, err, "s3 is deliberately out of scope until its config shape is modelled")
		assert.Contains(t, err.Error(), "invalid source_definition")
	})

	t.Run("rejects a duplicate id", func(t *testing.T) {
		h := table.NewHandler(&mockStore{}, t.TempDir())
		require.NoError(t, h.LoadSpec("a.yaml", validSpec()))
		require.Error(t, h.LoadSpec("b.yaml", validSpec()))
	})
}

func TestCreateSendsTableSourceType(t *testing.T) {
	var got *retlClient.RETLSourceCreateRequest
	store := &mockStore{
		createFunc: func(_ context.Context, s *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
			got = s
			return &retlClient.RETLSource{
				ID:                   "src-1",
				Name:                 s.Name,
				Config:               s.Config,
				SourceType:           s.SourceType,
				SourceDefinitionName: s.SourceDefinitionName,
				AccountID:            s.AccountID,
				IsEnabled:            s.Enabled,
			}, nil
		},
	}
	h := table.NewHandler(store, t.TempDir())
	require.NoError(t, h.LoadSpec("spec.yaml", validSpec()))
	res, err := h.GetResources()
	require.NoError(t, err)

	out, err := h.Create(context.Background(), "users-table", res[0].Data())
	require.NoError(t, err)

	require.NotNil(t, got)
	assert.Equal(t, retlClient.TableSourceType, got.SourceType, "must not be sent as a model source")
	assert.Equal(t, "users-table", got.ExternalID, "external id carries the local URN for import matching")
	cfg, ok := got.Config.(retlClient.RETLTableConfig)
	require.True(t, ok, "config must be the table config shape")
	assert.Equal(t, "public", cfg.Schema)
	assert.Equal(t, "users", cfg.Table)
	assert.Equal(t, "id", cfg.PrimaryKey)

	assert.Equal(t, "src-1", (*out)[table.IDKey])
}

func TestUpdateRejectsSourceDefinitionChange(t *testing.T) {
	h := table.NewHandler(&mockStore{}, t.TempDir())
	require.NoError(t, h.LoadSpec("spec.yaml", validSpec()))
	res, err := h.GetResources()
	require.NoError(t, err)

	state := resources.ResourceData{
		table.IDKey:               "src-1",
		table.SourceDefinitionKey: "postgres", // differs from the spec's snowflake
	}

	_, err = h.Update(context.Background(), "users-table", res[0].Data(), state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source definition name cannot be changed")
}

// Preview is meaningful for a SQL model and meaningless for a table — the
// webapp's table flow has no preview step either. Asserted so the difference is
// deliberate rather than an oversight.
func TestPreviewUnsupported(t *testing.T) {
	h := table.NewHandler(&mockStore{}, t.TempDir())
	_, err := h.Preview(context.Background(), "users-table", resources.ResourceData{}, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestDiffUpstreamDetectsTableMove(t *testing.T) {
	local := &table.TableResource{
		DisplayName: "Users", AccountID: "acc-123", PrimaryKey: "id",
		Schema: "public", Table: "users", Enabled: true,
	}
	same := *local
	assert.False(t, local.DiffUpstream(&same))

	movedSchema := *local
	movedSchema.Schema = "analytics"
	assert.True(t, local.DiffUpstream(&movedSchema), "a schema move must be seen as a change")

	movedTable := *local
	movedTable.Table = "users_v2"
	assert.True(t, local.DiffUpstream(&movedTable), "a table rename must be seen as a change")
}
