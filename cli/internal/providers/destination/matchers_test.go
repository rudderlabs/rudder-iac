package destination

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localDestination(id, displayName, destType string) *resources.Resource {
	return resources.NewResource(id, DestinationResourceType, resources.ResourceData{}, []string{},
		resources.WithRawData(&DestinationResource{ID: id, DisplayName: displayName, Type: destType}))
}

func remoteDestination(remoteID, name string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:         remoteID,
		ExternalID: "imported-1",
		Reference:  "#destination:imported-1",
		Data:       &RemoteDestination{Destination: &client.Destination{ID: remoteID, Name: name}},
	}
}

func destinationScope(rs ...*resources.Resource) importmatcher.Scope {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return importmatcher.Scope{LocalGraph: g}
}

func TestDestinationMatcher(t *testing.T) {
	t.Parallel()

	matchers := NewProvider(nil, nil).ResourceMatchers()
	require.Len(t, matchers, 1)
	require.Equal(t, DestinationResourceType, matchers[0].ResourceType)
	matcher := matchers[0]

	t.Run("matches on display name", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", "Prod Warehouse", "s3"))

		local := matcher.Match(scope, remoteDestination("dest-1", "Prod Warehouse"))

		require.NotNil(t, local)
		assert.Equal(t, "warehouse", local.ID())
	})

	t.Run("no match for different display name", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", "Prod Warehouse", "s3"))

		assert.Nil(t, matcher.Match(scope, remoteDestination("dest-1", "Staging Warehouse")))
	})

	t.Run("no match for a case-variant display name", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", "Prod Warehouse", "s3"))

		assert.Nil(t, matcher.Match(scope, remoteDestination("dest-1", "prod warehouse")))
	})

	t.Run("matches regardless of type", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", "Prod Warehouse", "postgres"))

		local := matcher.Match(scope, remoteDestination("dest-1", "Prod Warehouse"))

		require.NotNil(t, local)
		assert.Equal(t, "warehouse", local.ID())
	})

	t.Run("empty name never matches", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", "", "s3"))

		assert.Nil(t, matcher.Match(scope, remoteDestination("dest-1", "")))
	})
}
