package sqlmodel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

func TestParseAccountRef(t *testing.T) {
	t.Parallel()

	id, err := sqlmodel.ParseAccountRef("#account:prod-pg")
	require.NoError(t, err)
	assert.Equal(t, "prod-pg", id)

	for _, ref := range []string{"prod-pg", "#account:", "#destination:prod-pg", "account:prod-pg", "#account:prod\npg"} {
		_, err := sqlmodel.ParseAccountRef(ref)
		assert.Error(t, err, ref)
	}
}

// modelSpec is a SQL model spec on the legacy version existing users are on,
// naming its account with the given field.
func modelSpec(accountKey, account string) *specs.Spec {
	return createTestSpecMap(map[string]any{
		"id":                "orders",
		"display_name":      "Orders",
		"sql":               "SELECT 1",
		"primary_key":       "id",
		"source_definition": "postgres",
		accountKey:          account,
	})
}

func TestLoadSpecAccount(t *testing.T) {
	t.Parallel()

	t.Run("reference becomes a PropertyRef under account_id", func(t *testing.T) {
		t.Parallel()
		h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")
		require.NoError(t, h.LoadSpec("orders.yaml", modelSpec("account", "#account:prod-pg")))

		got, err := h.GetResources()
		require.NoError(t, err)
		require.Len(t, got, 1)
		ref, ok := got[0].Data()[sqlmodel.AccountIDKey].(*resources.PropertyRef)
		require.True(t, ok, "account_id holds a reference, got %T", got[0].Data()[sqlmodel.AccountIDKey])
		assert.Equal(t, "account:prod-pg", ref.URN)
		assert.Equal(t, "id", ref.Property)
	})

	t.Run("account_id stays a plain id", func(t *testing.T) {
		t.Parallel()
		h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")
		require.NoError(t, h.LoadSpec("orders.yaml", modelSpec("account_id", "acc-1")))

		got, err := h.GetResources()
		require.NoError(t, err)
		assert.Equal(t, "acc-1", got[0].Data()[sqlmodel.AccountIDKey])
	})

	invalid := []struct {
		name    string
		spec    *specs.Spec
		wantErr string
	}{
		{
			name:    "neither account_id nor account",
			spec:    modelSpec("description", ""),
			wantErr: "account_id or account must be specified",
		},
		{
			name: "both account_id and account",
			spec: func() *specs.Spec {
				s := modelSpec("account", "#account:prod-pg")
				s.Spec["account_id"] = "acc-1"
				return s
			}(),
			wantErr: "account_id and account cannot be specified together",
		},
		{
			name:    "account that is not an account reference",
			spec:    modelSpec("account", "prod-pg"),
			wantErr: `invalid account reference "prod-pg"`,
		},
	}
	for _, tc := range invalid {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			t.Parallel()
			h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")

			err := h.LoadSpec("orders.yaml", tc.spec)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// Remote state writes the account in the form the local spec uses, so neither
// form shows a change once applied — in particular an existing account_id spec
// whose account the CLI has since started managing.
func TestMapRemoteToStateAccountForm(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		accountKey string
		account    string
	}{
		{name: "reference to a managed account", accountKey: "account", account: "#account:prod-pg"},
		{name: "account_id of a managed account", accountKey: "account_id", account: "acc-remote"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")
			require.NoError(t, h.LoadSpec("orders.yaml", modelSpec(tc.accountKey, tc.account)))
			local, err := h.GetResources()
			require.NoError(t, err)

			collection := resources.NewRemoteResources()
			collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
				"src-1": {ID: "src-1", ExternalID: "orders", Data: retlClient.RETLSource{
					ID: "src-1", Name: "Orders", ExternalID: "orders", IsEnabled: true,
					SourceType: retlClient.ModelSourceType, SourceDefinitionName: "postgres", AccountID: "acc-remote",
					Config: retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: "SELECT 1"},
				}},
			})
			collection.Set(accounts.AccountResourceType, map[string]*resources.RemoteResource{
				"acc-remote": {ID: "acc-remote", ExternalID: "prod-pg"},
			})

			st, err := h.MapRemoteToState(collection)
			require.NoError(t, err)
			rs := st.GetResource(local[0].URN())
			require.NotNil(t, rs)

			diffs, _ := differ.CompareData(rs.Input, local[0].Data())
			assert.Empty(t, diffs)
		})
	}
}

func TestPreviewReferencedAccount(t *testing.T) {
	t.Parallel()
	h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")

	_, err := h.Preview(context.Background(), "orders", resources.ResourceData{
		sqlmodel.SQLKey:       "SELECT 1",
		sqlmodel.AccountIDKey: sqlmodel.AccountRef("prod-pg"),
	}, 5)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "set account_id on orders to preview it")
}
