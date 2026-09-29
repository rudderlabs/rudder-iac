package sqlmodel_test

import (
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopeWith(rs ...*resources.Resource) importmatcher.Scope {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return importmatcher.Scope{LocalGraph: g}
}

func localModel(id, displayName, accountID string) *resources.Resource {
	return resources.NewResource(id, sqlmodel.ResourceType, resources.ResourceData{
		sqlmodel.DisplayNameKey: displayName,
		sqlmodel.AccountIDKey:   accountID,
	}, []string{})
}

func remoteModel(remoteID, name, accountID string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:   remoteID,
		Data: &retlClient.RETLSource{ID: remoteID, Name: name, AccountID: accountID},
	}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	m := sqlmodel.Matcher()
	assert.Equal(t, sqlmodel.ResourceType, m.ResourceType)

	t.Run("matches on display name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("orders", "Orders", "acc_1"))

		local := m.Match(scope, remoteModel("src_1", "Orders", "acc_1"))

		require.NotNil(t, local)
		assert.Equal(t, "orders", local.ID())
	})

	t.Run("matches a name differing only in case, as the server does", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("orders", "orders", "acc_1"))

		local := m.Match(scope, remoteModel("src_1", "Orders", "acc_1"))

		require.NotNil(t, local)
		assert.Equal(t, "orders", local.ID())
	})

	t.Run("no match for different display name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("orders", "Orders", "acc_1"))

		assert.Nil(t, m.Match(scope, remoteModel("src_1", "Customers", "acc_1")))
	})

	// Names are unique across the workspace regardless of account, so the same
	// name in a different account is the same source — one whose account the
	// local spec changes. Refusing to link it wrote a duplicate spec instead.
	t.Run("matches even when the account id differs", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("orders", "Orders", "acc_1"))

		local := m.Match(scope, remoteModel("src_1", "Orders", "acc_2"))

		require.NotNil(t, local)
		assert.Equal(t, "orders", local.ID())
	})

	t.Run("empty display name never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("broken", "", "acc_1"))

		assert.Nil(t, m.Match(scope, remoteModel("src_1", "", "acc_1")))
	})

	// The regression: a PropertyRef account has no remote id until apply, so
	// while account_id was a second conjunct this spec could never match its own
	// remote twin, and import --merge wrote a second spec for it.
	t.Run("matches a model that references its account", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(resources.NewResource("orders", sqlmodel.ResourceType, resources.ResourceData{
			sqlmodel.DisplayNameKey: "Orders",
			sqlmodel.AccountIDKey:   sqlmodel.AccountRef("prod-pg"),
		}, []string{}))

		local := m.Match(scope, remoteModel("src_1", "Orders", "acc_1"))

		require.NotNil(t, local)
		assert.Equal(t, "orders", local.ID())
	})
}
