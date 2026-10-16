package table_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

func scopeWith(rs ...*resources.Resource) importmatcher.Scope {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return importmatcher.Scope{LocalGraph: g}
}

func localSource(id, resourceType, displayName, accountID string) *resources.Resource {
	return resources.NewResource(id, resourceType, resources.ResourceData{
		sqlmodel.DisplayNameKey: displayName,
		sqlmodel.AccountIDKey:   accountID,
	}, []string{})
}

func remoteSource(remoteID, name, accountID string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:   remoteID,
		Data: &retlClient.RETLSource{ID: remoteID, Name: name, AccountID: accountID},
	}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	m := table.Matcher()
	assert.Equal(t, table.ResourceType, m.ResourceType)

	t.Run("matches on display name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("users-table", table.ResourceType, "Users", "acc-1"))

		local := m.Match(scope, remoteSource("src-1", "Users", "acc-1"))

		require.NotNil(t, local)
		assert.Equal(t, "users-table", local.ID())
	})

	t.Run("matches a name differing only in case, as the server does", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("users-table", table.ResourceType, "users", "acc-1"))

		local := m.Match(scope, remoteSource("src-1", "Users", "acc-1"))

		require.NotNil(t, local)
		assert.Equal(t, "users-table", local.ID())
	})

	// See sqlmodel's matcher test: names are workspace-unique regardless of
	// account, so a differing account is a change to apply, not a different source.
	t.Run("matches even when the account id differs", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("users-table", table.ResourceType, "Users", "acc-1"))

		local := m.Match(scope, remoteSource("src-1", "Users", "acc-2"))

		require.NotNil(t, local)
		assert.Equal(t, "users-table", local.ID())
	})

	t.Run("matches a table source that references its account", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(resources.NewResource("users-table", table.ResourceType, resources.ResourceData{
			sqlmodel.DisplayNameKey: "Users",
			sqlmodel.AccountIDKey:   sqlmodel.AccountRef("prod-pg"),
		}, []string{}))

		local := m.Match(scope, remoteSource("src-1", "Users", "acc-1"))

		require.NotNil(t, local)
		assert.Equal(t, "users-table", local.ID())
	})

	t.Run("never matches a SQL model of the same name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("users-model", sqlmodel.ResourceType, "Users", "acc-1"))

		assert.Nil(t, m.Match(scope, remoteSource("src-1", "Users", "acc-1")))
	})

	t.Run("empty display name never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("broken", table.ResourceType, "", "acc-1"))

		assert.Nil(t, m.Match(scope, remoteSource("src-1", "", "acc-1")))
	})
}
