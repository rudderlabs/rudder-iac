package resolver_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collectionWith(resourceType string, rs ...*resources.RemoteResource) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	m := make(map[string]*resources.RemoteResource, len(rs))
	for _, r := range rs {
		m[r.ID] = r
	}
	collection.Set(resourceType, m)
	return collection
}

func TestResolveToReference(t *testing.T) {
	t.Parallel()

	t.Run("importable resource resolves to its reference", func(t *testing.T) {
		t.Parallel()

		r := &resolver.ImportRefResolver{
			Remote: resources.NewRemoteResources(),
			Graph:  resources.NewGraph(),
			Importable: collectionWith("category", &resources.RemoteResource{
				ID:        "cat_1",
				Reference: "#category:checkout",
			}),
		}

		ref, err := r.ResolveToReference("category", "cat_1")

		require.NoError(t, err)
		assert.Equal(t, "#category:checkout", ref)
	})

	t.Run("managed resource missing from local graph is a pending-delete conflict", func(t *testing.T) {
		t.Parallel()

		// Managed remotely (externalID set) but absent from the local graph:
		// the spec was deleted locally and the deletion is not applied yet.
		r := &resolver.ImportRefResolver{
			Remote: collectionWith("category", &resources.RemoteResource{
				ID:         "cat_1",
				ExternalID: "checkout",
			}),
			Graph:      resources.NewGraph(),
			Importable: resources.NewRemoteResources(),
		}

		_, err := r.ResolveToReference("category", "cat_1")

		require.ErrorIs(t, err, resolver.ErrPendingDeleteConflict)
		assert.ErrorContains(t, err, "category:checkout")
		assert.ErrorContains(t, err, "cat_1")
	})

	t.Run("unknown remote id is not a pending-delete conflict", func(t *testing.T) {
		t.Parallel()

		r := &resolver.ImportRefResolver{
			Remote:     resources.NewRemoteResources(),
			Graph:      resources.NewGraph(),
			Importable: resources.NewRemoteResources(),
		}

		_, err := r.ResolveToReference("category", "cat_unknown")

		require.Error(t, err)
		assert.NotErrorIs(t, err, resolver.ErrPendingDeleteConflict)
	})
}
