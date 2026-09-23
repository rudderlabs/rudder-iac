package source_test

import (
	"testing"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
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

func localSource(id, name string) *resources.Resource {
	return resources.NewResource(id, source.ResourceType, resources.ResourceData{source.NameKey: name}, []string{})
}

func remoteSource(remoteID, name string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:   remoteID,
		Data: &sourceClient.EventStreamSource{ID: remoteID, Name: name},
	}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	m := source.Matcher()
	assert.Equal(t, source.ResourceType, m.ResourceType)

	t.Run("matches on name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("web-source", "Web Source"))

		local := m.Match(scope, remoteSource("src_1", "Web Source"))

		require.NotNil(t, local)
		assert.Equal(t, "web-source", local.ID())
	})

	t.Run("no match for different name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("web-source", "Web Source"))

		assert.Nil(t, m.Match(scope, remoteSource("src_1", "Mobile Source")))
	})

	t.Run("empty name never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localSource("broken", ""))

		assert.Nil(t, m.Match(scope, remoteSource("src_1", "")))
	})
}
