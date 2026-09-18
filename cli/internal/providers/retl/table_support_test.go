package retl_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// workspaceSources is one managed and one unmanaged source of each kind.
var workspaceSources = []retlClient.RETLSource{
	{
		ID: "model-managed", Name: "Orders Model", ExternalID: "orders-model", WorkspaceID: "ws-1",
		SourceType: retlClient.ModelSourceType, SourceDefinitionName: "postgres", AccountID: "acc-1",
		Config: retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: "SELECT 1"},
	},
	{
		ID: "model-unmanaged", Name: "Users Model", WorkspaceID: "ws-1",
		SourceType: retlClient.ModelSourceType, SourceDefinitionName: "postgres", AccountID: "acc-1",
		Config: retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: "SELECT 2"},
	},
	{
		ID: "table-managed", Name: "Users", ExternalID: "users-table", WorkspaceID: "ws-1",
		SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres", AccountID: "acc-1",
		Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
	},
	{
		ID: "table-unmanaged", Name: "Raw Events", WorkspaceID: "ws-1",
		SourceType: retlClient.TableSourceType, SourceDefinitionName: "s3", AccountID: "acc-2",
		Config: retlClient.RETLS3TableConfig{BucketName: "events"},
	},
}

// newWorkspaceClient returns a mock serving workspaceSources, honouring the
// sourceType and hasExternalId filters as the API does, and the list of
// sourceType filters it was called with.
func newWorkspaceClient() (*mockRETLStore, *[]string) {
	var sourceTypes []string
	client := newDefaultMockClient()
	client.listRetlSourcesFunc = func(_ context.Context, opts ...retlClient.ListRetlSourcesOption) (*retlClient.RETLSources, error) {
		var o retlClient.ListRetlSourcesOptions
		for _, opt := range opts {
			opt(&o)
		}
		sourceTypes = append(sourceTypes, o.SourceType)

		result := &retlClient.RETLSources{}
		for _, s := range workspaceSources {
			if string(s.SourceType) != o.SourceType {
				continue
			}
			if o.HasExternalId != nil && (s.ExternalID != "") != *o.HasExternalId {
				continue
			}
			result.Data = append(result.Data, s)
		}
		return result, nil
	}
	return client, &sourceTypes
}

func sqlModelPatterns() []vrules.MatchPattern {
	var want []vrules.MatchPattern
	want = append(want, prules.LegacyVersionPatterns(sqlmodel.ResourceKind)...)
	want = append(want, prules.V1VersionPatterns(sqlmodel.ResourceKind)...)
	return want
}

func tableSpec() *specs.Spec {
	return &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    table.ResourceKind,
		Spec: map[string]any{
			"id":                "users-table",
			"display_name":      "Users",
			"account_id":        "acc-1",
			"source_definition": "postgres",
			"primary_key":       "id",
			"schema":            "public",
			"table":             "users",
		},
	}
}

// Without WithTableSupport the provider must look exactly as it did before the
// kind existed: the flag is off by default, so this is what every user gets.
func TestTableSupportDisabled(t *testing.T) {
	t.Parallel()

	t.Run("exposes only the SQL model kind", func(t *testing.T) {
		t.Parallel()
		p := retl.New(newDefaultMockClient())

		assert.Equal(t, []string{sqlmodel.ResourceKind}, p.SupportedKinds())
		assert.Equal(t, []string{sqlmodel.ResourceType}, p.SupportedTypes())
		assert.ElementsMatch(t, sqlModelPatterns(), p.SupportedMatchPatterns())

		matchers := p.ResourceMatchers()
		require.Len(t, matchers, 1)
		assert.Equal(t, sqlmodel.ResourceType, matchers[0].ResourceType)

		assert.Equal(t, []string{"retl/sqlmodel/spec-syntax-valid"}, ruleIDs(p.SyntacticRules()))
		assert.Equal(t, []string{"retl/sqlmodel/semantic-valid"}, ruleIDs(p.SemanticRules()))
	})

	t.Run("rejects table specs as an unknown kind", func(t *testing.T) {
		t.Parallel()
		p := retl.New(newDefaultMockClient())

		_, err := p.ParseSpec("users.yaml", tableSpec())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported kind: retl-source-table")

		err = p.LoadSpec("users.yaml", tableSpec())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported kind: retl-source-table")

		_, err = p.Create(context.Background(), "users-table", table.ResourceType, resources.ResourceData{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no handler for resource type: retl-source-table")

		_, err = p.Preview(context.Background(), "users-table", table.ResourceType, resources.ResourceData{}, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no handler for resource type: retl-source-table")
	})

	t.Run("never lists table sources", func(t *testing.T) {
		t.Parallel()
		client, sourceTypes := newWorkspaceClient()
		p := retl.New(client)
		ctx := context.Background()

		remote, err := p.LoadResourcesFromRemote(ctx)
		require.NoError(t, err)
		assert.Empty(t, remote.GetAll(table.ResourceType))

		importable, err := p.LoadImportable(ctx, namer.NewExternalIdNamer(namer.StrategyKebabCase))
		require.NoError(t, err)
		assert.Empty(t, importable.GetAll(table.ResourceType))

		_, err = p.List(ctx, table.ResourceType, lister.Filters{})
		require.Error(t, err)

		assert.Equal(t, []string{string(retlClient.ModelSourceType), string(retlClient.ModelSourceType)}, *sourceTypes)
	})
}

func ruleIDs(rs []vrules.Rule) []string {
	ids := make([]string, 0, len(rs))
	for _, r := range rs {
		ids = append(ids, r.ID())
	}
	return ids
}

func TestTableSupportEnabled(t *testing.T) {
	t.Parallel()

	t.Run("adds the table semantic rule", func(t *testing.T) {
		t.Parallel()
		p := retl.New(newDefaultMockClient(), retl.WithTableSupport())

		assert.Equal(t, []string{"retl/sqlmodel/semantic-valid", "retl/table/semantic-valid"}, ruleIDs(p.SemanticRules()))
	})

	t.Run("adds the table kind on v1 only", func(t *testing.T) {
		t.Parallel()
		p := retl.New(newDefaultMockClient(), retl.WithTableSupport())

		assert.ElementsMatch(t, []string{sqlmodel.ResourceKind, table.ResourceKind}, p.SupportedKinds())
		assert.ElementsMatch(t, []string{sqlmodel.ResourceType, table.ResourceType}, p.SupportedTypes())

		// The SQL model kind keeps its legacy patterns; the table kind was
		// introduced after legacy versions were retired, and letting a project
		// pin rudder/0.1 on it would be a support promise to withdraw later.
		want := append(sqlModelPatterns(), prules.V1VersionPatterns(table.ResourceKind)...)
		assert.ElementsMatch(t, want, p.SupportedMatchPatterns())
		for _, legacy := range prules.LegacyVersionPatterns(table.ResourceKind) {
			assert.NotContains(t, p.SupportedMatchPatterns(), legacy)
		}
	})

	t.Run("registers the table syntax rule", func(t *testing.T) {
		t.Parallel()
		p := retl.New(newDefaultMockClient(), retl.WithTableSupport())

		syntactic := p.SyntacticRules()
		require.Len(t, syntactic, 2)
		assert.Equal(t, "retl/table/spec-syntax-valid", syntactic[1].ID())
	})

	t.Run("orders source matchers SQL model first", func(t *testing.T) {
		t.Parallel()
		p := retl.New(newDefaultMockClient(), retl.WithTableSupport())

		matchers := p.ResourceMatchers()

		require.Len(t, matchers, 2)
		assert.Equal(t, sqlmodel.ResourceType, matchers[0].ResourceType)
		assert.Equal(t, table.ResourceType, matchers[1].ResourceType)
	})

	// import --merge matches in this order, and a connection resolves its
	// endpoints through source matches. retlOptions happens to pass the
	// connection option first; the order must not depend on that.
	t.Run("orders the connection matcher after every source matcher", func(t *testing.T) {
		t.Parallel()
		want := []string{sqlmodel.ResourceType, table.ResourceType, connection.ResourceType}

		connectionFirst := retl.New(newDefaultMockClient(), retl.WithConnectionSupport(nil), retl.WithTableSupport())
		tableFirst := retl.New(newDefaultMockClient(), retl.WithTableSupport(), retl.WithConnectionSupport(nil))

		assert.Equal(t, want, matcherTypes(connectionFirst.ResourceMatchers()))
		assert.Equal(t, want, matcherTypes(tableFirst.ResourceMatchers()))
	})

	t.Run("loads table specs into the resource graph", func(t *testing.T) {
		t.Parallel()
		p := retl.New(newDefaultMockClient(), retl.WithTableSupport())

		parsed, err := p.ParseSpec("users.yaml", tableSpec())
		require.NoError(t, err)
		assert.Equal(t, "retl-source-table:users-table", parsed.URNs[0].URN)

		require.NoError(t, p.LoadSpec("users.yaml", tableSpec()))
		require.NoError(t, p.LoadImportManifest(&specs.WorkspaceImportMetadata{
			WorkspaceID: "ws-1",
			Resources:   []specs.ImportIds{{URN: "retl-source-table:users-table", RemoteID: "table-managed"}},
		}))

		graph, err := p.ResourceGraph()
		require.NoError(t, err)
		r, ok := graph.GetResource("retl-source-table:users-table")
		require.True(t, ok)
		assert.Equal(t, "Users", r.Data()[sqlmodel.DisplayNameKey])
		require.NotNil(t, r.ImportMetadata(), "the import manifest reaches the table handler")
		assert.Equal(t, "table-managed", r.ImportMetadata().RemoteId)
	})

	t.Run("reads remote state for both source kinds", func(t *testing.T) {
		t.Parallel()
		client, sourceTypes := newWorkspaceClient()
		p := retl.New(client, retl.WithTableSupport())

		remote, err := p.LoadResourcesFromRemote(context.Background())
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{string(retlClient.ModelSourceType), string(retlClient.TableSourceType)}, *sourceTypes)

		st, err := p.MapRemoteToState(remote)
		require.NoError(t, err)
		require.NotNil(t, st.GetResource("retl-source-sql-model:orders-model"))
		users := st.GetResource("retl-source-table:users-table")
		require.NotNil(t, users)
		assert.Equal(t, "table-managed", users.Output[sqlmodel.IDKey])
		assert.Len(t, st.Resources, 2)
	})

	t.Run("exports unmanaged sources of both kinds", func(t *testing.T) {
		t.Parallel()
		client, _ := newWorkspaceClient()
		p := retl.New(client, retl.WithTableSupport())

		importable, err := p.LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.StrategyKebabCase))
		require.NoError(t, err)

		entities, entries, err := p.FormatForExport(importable, namer.NewExternalIdNamer(namer.StrategyKebabCase), noopResolver{})
		require.NoError(t, err)

		paths := make([]string, 0, len(entities))
		for _, e := range entities {
			paths = append(paths, e.RelativePath)
		}
		assert.ElementsMatch(t, []string{"retl/sql-models/users-model.yaml", "retl/tables/raw-events.yaml"}, paths)
		assert.Len(t, entries, 2)
	})

	t.Run("creates table sources through the provider", func(t *testing.T) {
		t.Parallel()
		client := newDefaultMockClient()
		var got *retlClient.RETLSourceCreateRequest
		client.createRetlSourceFunc = func(_ context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
			got = req
			return &retlClient.RETLSource{
				ID: "src-new", Name: req.Name, Config: req.Config, SourceType: req.SourceType,
				SourceDefinitionName: req.SourceDefinitionName, AccountID: req.AccountID, IsEnabled: req.Enabled,
			}, nil
		}
		p := retl.New(client, retl.WithTableSupport())
		require.NoError(t, p.LoadSpec("users.yaml", tableSpec()))
		graph, err := p.ResourceGraph()
		require.NoError(t, err)
		r, ok := graph.GetResource("retl-source-table:users-table")
		require.True(t, ok)

		output, err := p.Create(context.Background(), r.ID(), r.Type(), r.Data())

		require.NoError(t, err)
		assert.Equal(t, &retlClient.RETLSourceCreateRequest{
			Name:                 "Users",
			Config:               retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
			SourceType:           retlClient.TableSourceType,
			SourceDefinitionName: "postgres",
			AccountID:            "acc-1",
			Enabled:              true,
			ExternalID:           "users-table",
		}, got)
		assert.Equal(t, "src-new", (*output)[sqlmodel.IDKey])
	})

	// What this pins that the handler-level tests cannot: that Provider.Preview
	// forwards its limit rather than dropping or hardcoding it. preview_test.go
	// calls the handler directly, so a Provider.Preview that passed a literal
	// would still pass there. The handler-map dispatch itself is covered by the
	// create subtest above, which reads the same p.handlers map.
	t.Run("validates table sources through the preview API", func(t *testing.T) {
		t.Parallel()
		client := newDefaultMockClient()
		var got *retlClient.PreviewSubmitRequest
		client.submitPreviewFunc = func(_ context.Context, req *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
			got = req
			return &retlClient.PreviewSubmitResponse{ID: "req-1"}, nil
		}
		p := retl.New(client, retl.WithTableSupport())
		require.NoError(t, p.LoadSpec("users.yaml", tableSpec()))
		graph, err := p.ResourceGraph()
		require.NoError(t, err)
		r, ok := graph.GetResource("retl-source-table:users-table")
		require.True(t, ok)

		_, err = p.Preview(context.Background(), r.ID(), r.Type(), r.Data(), 0)

		require.NoError(t, err)
		assert.Equal(t, &retlClient.PreviewSubmitRequest{AccountID: "acc-1", SQL: `select * from "public"."users" limit 1`}, got)
	})
}
