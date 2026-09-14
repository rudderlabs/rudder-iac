package table_test

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// referencing swaps a spec's account_id for a reference to the account.
func referencing(s *specs.Spec, account string) *specs.Spec {
	return withField(withoutField(s, "account_id"), "account", "#account:"+account)
}

func TestAccountReference(t *testing.T) {
	t.Parallel()

	t.Run("becomes a dependency on the account that resolves to its remote id", func(t *testing.T) {
		t.Parallel()
		store := newFakeStore()
		h, r := loadResource(t, store, referencing(warehouseSpec(), "prod-pg"))

		ref, ok := r.Data()[sqlmodel.AccountIDKey].(*resources.PropertyRef)
		require.True(t, ok, "account_id holds a reference, got %T", r.Data()[sqlmodel.AccountIDKey])
		assert.Equal(t, "account:prod-pg", ref.URN)
		assert.Equal(t, "id", ref.Property)

		graph := resources.NewGraph()
		graph.AddResource(r)
		assert.Equal(t, []string{"account:prod-pg"}, graph.GetDependencies(r.URN()))

		// The syncer dereferences against state before Create; an account's
		// state is the accounts provider's typed AccountState.
		st := state.EmptyState()
		st.AddResource(&state.ResourceState{
			ID:        "prod-pg",
			Type:      accounts.AccountResourceType,
			OutputRaw: &accounts.AccountState{ID: "acc-remote"},
		})
		data, err := state.Dereference(r.Data(), st)
		require.NoError(t, err)
		assert.Equal(t, "acc-remote", data[sqlmodel.AccountIDKey])

		output, err := h.Create(context.Background(), r.ID(), data)
		require.NoError(t, err)
		assert.Equal(t, "acc-remote", store.sources[(*output)[sqlmodel.IDKey].(string)].AccountID)
	})

	// The handler takes an s3 reference as it does a warehouse one; whether the
	// account can back s3 is the semantic rule's call.
	t.Run("s3 source references its account too", func(t *testing.T) {
		t.Parallel()
		_, r := loadResource(t, newFakeStore(), referencing(s3Spec(), "events-s3"))

		ref, ok := r.Data()[sqlmodel.AccountIDKey].(*resources.PropertyRef)
		require.True(t, ok)
		assert.Equal(t, "account:events-s3", ref.URN)
	})
}

// Remote state writes the account in the form the local spec uses, so a plan
// right after apply is empty for either form.
func TestAccountStateFollowsSpecForm(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		spec            *specs.Spec
		managed         bool
		wantAccountDiff bool
	}{
		{
			name:    "reference to a managed account",
			spec:    referencing(warehouseSpec(), "prod-pg"),
			managed: true,
		},
		{
			// An existing spec keeps its raw id once the CLI manages the
			// account, and must not show a change on every plan.
			name:    "account_id of a managed account",
			spec:    withField(warehouseSpec(), "account_id", "acc-remote"),
			managed: true,
		},
		{
			// The source points at an account the CLI does not manage, so
			// apply moves it to the referenced one.
			name:            "reference while the remote account is unmanaged",
			spec:            referencing(warehouseSpec(), "prod-pg"),
			wantAccountDiff: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			h, r := loadResource(t, newFakeStore(), tc.spec)

			created := maps.Clone(r.Data())
			created[sqlmodel.AccountIDKey] = "acc-remote"
			_, err := h.Create(ctx, r.ID(), created)
			require.NoError(t, err)

			collection, err := h.LoadResourcesFromRemote(ctx)
			require.NoError(t, err)
			if tc.managed {
				collection.Set(accounts.AccountResourceType, map[string]*resources.RemoteResource{
					"acc-remote": {ID: "acc-remote", ExternalID: "prod-pg"},
				})
			}
			st, err := h.MapRemoteToState(collection)
			require.NoError(t, err)
			rs := st.GetResource(r.URN())
			require.NotNil(t, rs)

			diffs, _ := differ.CompareData(rs.Input, r.Data())
			if !tc.wantAccountDiff {
				assert.Empty(t, diffs)
				return
			}
			require.Len(t, diffs, 1)
			assert.Contains(t, diffs, sqlmodel.AccountIDKey)
		})
	}
}

// importResolver adds accounts (remote id to local id) to the import set, and
// returns the resolver import workspace builds over it.
func importResolver(importable *resources.RemoteResources, accountIDs map[string]string) *resolver.ImportRefResolver {
	importableAccounts := make(map[string]*resources.RemoteResource, len(accountIDs))
	for remoteID, localID := range accountIDs {
		importableAccounts[remoteID] = &resources.RemoteResource{ID: remoteID, ExternalID: localID, Reference: "#account:" + localID}
	}
	importable.Set(accounts.AccountResourceType, importableAccounts)
	return &resolver.ImportRefResolver{
		Remote:     resources.NewRemoteResources(),
		Graph:      resources.NewGraph(),
		Importable: importable,
	}
}

// Import writes the account in a form that loads back and, once apply has
// adopted the source and the account imported with it, plans no change.
func TestExportAccountRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		accountIDs     map[string]string
		wantAccount    map[string]any
		accountManaged bool
	}{
		{
			name:           "references an account imported alongside",
			accountIDs:     map[string]string{"acc-remote": "prod-pg"},
			wantAccount:    map[string]any{"account": "#account:prod-pg"},
			accountManaged: true,
		},
		{
			name:        "falls back to account_id for an account it cannot resolve",
			wantAccount: map[string]any{"account_id": "acc-remote"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			source := retlClient.RETLSource{
				ID: "src-1", Name: "Users", WorkspaceID: "ws-1", AccountID: "acc-remote", IsEnabled: true,
				SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres",
				Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
			}
			h := table.NewHandler(newFakeStore(source), "retl")
			importable, err := h.LoadImportable(ctx, namer.NewExternalIdNamer(namer.StrategyKebabCase))
			require.NoError(t, err)

			entities, _, err := h.FormatForExport(importable, nil, importResolver(importable, tc.accountIDs))
			require.NoError(t, err)
			require.Len(t, entities, 1)
			spec, ok := entities[0].Content.(*specs.Spec)
			require.True(t, ok)
			want := map[string]any{
				"id":                "users",
				"display_name":      "Users",
				"source_definition": "postgres",
				"primary_key":       "id",
				"schema":            "public",
				"table":             "users",
				"enabled":           true,
			}
			maps.Copy(want, tc.wantAccount)
			assert.Equal(t, want, spec.Spec)

			adopted := source
			adopted.ExternalID = "users"
			loaded, r := loadResource(t, newFakeStore(adopted), spec)
			collection, err := loaded.LoadResourcesFromRemote(ctx)
			require.NoError(t, err)
			if tc.accountManaged {
				collection.Set(accounts.AccountResourceType, map[string]*resources.RemoteResource{
					"acc-remote": {ID: "acc-remote", ExternalID: "prod-pg"},
				})
			}
			st, err := loaded.MapRemoteToState(collection)
			require.NoError(t, err)
			rs := st.GetResource(r.URN())
			require.NotNil(t, rs)

			diffs, _ := differ.CompareData(rs.Input, r.Data())
			assert.Empty(t, diffs)
		})
	}
}

// A source whose spec is gone is about to be deleted. Referencing its managed
// account in state orders that delete before the account's.
func TestAccountStateWithoutLocalSpec(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := table.NewHandler(newFakeStore(retlClient.RETLSource{
		ID: "src-1", Name: "Users", ExternalID: "users-table", WorkspaceID: "ws-1",
		SourceType: retlClient.TableSourceType, SourceDefinitionName: "postgres", AccountID: "acc-remote",
		Config: retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
	}), "retl")

	collection, err := h.LoadResourcesFromRemote(ctx)
	require.NoError(t, err)
	collection.Set(accounts.AccountResourceType, map[string]*resources.RemoteResource{
		"acc-remote": {ID: "acc-remote", ExternalID: "prod-pg"},
	})
	st, err := h.MapRemoteToState(collection)
	require.NoError(t, err)

	graph := syncer.StateToGraph(st)
	assert.Equal(t, []string{"account:prod-pg"}, graph.GetDependencies("retl-source-table:users-table"))
}
