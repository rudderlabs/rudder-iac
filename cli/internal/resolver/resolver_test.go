package resolver

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportRefResolver_ResolveToReference(t *testing.T) {
	t.Parallel()

	newResolver := func(importable, remote map[string]*resources.RemoteResource, graph *resources.Graph) *ImportRefResolver {
		importableResources := resources.NewRemoteResources()
		importableResources.Set("source", importable)

		remoteResources := resources.NewRemoteResources()
		remoteResources.Set("source", remote)

		return &ImportRefResolver{
			Importable: importableResources,
			Remote:     remoteResources,
			Graph:      graph,
		}
	}

	t.Run("returns importable reference when present", func(t *testing.T) {
		r := newResolver(
			map[string]*resources.RemoteResource{
				"r-1": {ID: "r-1", Reference: "#/source/common/imported"},
			},
			nil,
			resources.NewGraph(),
		)

		ref, err := r.ResolveToReference("source", "r-1")
		require.NoError(t, err)
		assert.Equal(t, "#/source/common/imported", ref)
	})

	t.Run("returns error when remote resource is missing", func(t *testing.T) {
		r := newResolver(nil, nil, resources.NewGraph())

		_, err := r.ResolveToReference("source", "missing")
		require.Error(t, err)
		assert.EqualError(t, err, "resource not present in resources collection")
	})

	t.Run("returns error when graph resource is missing", func(t *testing.T) {
		r := newResolver(
			nil,
			map[string]*resources.RemoteResource{
				"remote-1": {ID: "remote-1", ExternalID: "ext-1"},
			},
			resources.NewGraph(),
		)

		_, err := r.ResolveToReference("source", "remote-1")
		require.Error(t, err)
		assert.EqualError(t, err, "resource not present in resources graph")
	})

	t.Run("returns error when file metadata is missing", func(t *testing.T) {
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("ext-1", "source", resources.ResourceData{}, nil))

		r := newResolver(
			nil,
			map[string]*resources.RemoteResource{
				"remote-1": {ID: "remote-1", ExternalID: "ext-1"},
			},
			graph,
		)

		_, err := r.ResolveToReference("source", "remote-1")
		require.Error(t, err)
		assert.EqualError(t, err, "file metadata on the graph resource is not present")
	})

	t.Run("returns error when metadata ref is empty", func(t *testing.T) {
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("ext-1", "source", resources.ResourceData{}, nil, resources.WithResourceFileMetadata("")))

		r := newResolver(
			nil,
			map[string]*resources.RemoteResource{
				"remote-1": {ID: "remote-1", ExternalID: "ext-1"},
			},
			graph,
		)

		_, err := r.ResolveToReference("source", "remote-1")
		require.Error(t, err)
		assert.EqualError(t, err, "file metadata on the graph resource is not present")
	})

	t.Run("returns metadata ref from graph resource", func(t *testing.T) {
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"ext-1",
			"source",
			resources.ResourceData{},
			nil,
			resources.WithResourceFileMetadata("#/source/common/ext-1"),
		))

		r := newResolver(
			nil,
			map[string]*resources.RemoteResource{
				"remote-1": {ID: "remote-1", ExternalID: "ext-1"},
			},
			graph,
		)

		ref, err := r.ResolveToReference("source", "remote-1")
		require.NoError(t, err)
		assert.Equal(t, "#/source/common/ext-1", ref)
	})
}
