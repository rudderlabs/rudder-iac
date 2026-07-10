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

	t.Run("matches on display name and account id", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("orders", "Orders", "acc_1"))

		local := m.Match(scope, remoteModel("src_1", "Orders", "acc_1"))

		require.NotNil(t, local)
		assert.Equal(t, "orders", local.ID())
	})

	t.Run("no match for different display name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("orders", "Orders", "acc_1"))

		assert.Nil(t, m.Match(scope, remoteModel("src_1", "Customers", "acc_1")))
	})

	t.Run("no match when account id differs", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("orders", "Orders", "acc_1"))

		assert.Nil(t, m.Match(scope, remoteModel("src_1", "Orders", "acc_2")))
	})

	t.Run("empty display name never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localModel("broken", "", "acc_1"))

		assert.Nil(t, m.Match(scope, remoteModel("src_1", "", "acc_1")))
	})
}
