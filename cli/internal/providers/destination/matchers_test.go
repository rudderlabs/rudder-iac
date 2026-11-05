package destination

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localDestination(id, displayName string) *resources.Resource {
	return resources.NewResource(id, DestinationResourceType, resources.ResourceData{}, []string{},
		resources.WithRawData(&DestinationResource{ID: id, DisplayName: displayName, Type: "s3"}))
}

// remoteDestination is a webhook, a different type from every local fixture.
func remoteDestination(remoteID, name string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:   remoteID,
		Data: &RemoteDestination{Destination: &client.Destination{ID: remoteID, Name: name, Type: "WEBHOOK"}},
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

	t.Run("matches on display name whatever the type", func(t *testing.T) {
		t.Parallel()

		warehouse := localDestination("warehouse", "Prod Warehouse")

		local := matcher.Match(destinationScope(warehouse), remoteDestination("dest-1", "Prod Warehouse"))

		assert.Same(t, warehouse, local)
	})

	t.Run("no match for different display name", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", "Prod Warehouse"))

		assert.Nil(t, matcher.Match(scope, remoteDestination("dest-1", "Staging Warehouse")))
	})

	t.Run("no match for a case-variant display name", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", "Prod Warehouse"))

		assert.Nil(t, matcher.Match(scope, remoteDestination("dest-1", "prod warehouse")))
	})

	t.Run("empty name never matches", func(t *testing.T) {
		t.Parallel()

		scope := destinationScope(localDestination("warehouse", ""))

		assert.Nil(t, matcher.Match(scope, remoteDestination("dest-1", "")))
	})
}
