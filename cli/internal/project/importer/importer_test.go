package importer

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSyncStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		diff    *differ.Diff
		merge   bool
		wantErr bool
	}{
		{
			name:  "no merge, synced project",
			diff:  &differ.Diff{},
			merge: false,
		},
		{
			name:    "no merge, pending changes block import",
			diff:    &differ.Diff{NewResources: []string{"category:checkout"}},
			merge:   false,
			wantErr: true,
		},
		{
			name:  "merge allows pending additions",
			diff:  &differ.Diff{NewResources: []string{"category:checkout"}},
			merge: true,
		},
		{
			name:    "merge still blocks pending deletions",
			diff:    &differ.Diff{RemovedResources: []string{"category:legacy"}},
			merge:   true,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkSyncStatus(tc.diff, tc.merge)

			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrProjectNotSynced)
		})
	}
}

type stubMatcherProvider struct {
	matchers []importmatcher.Matcher
}

func (s stubMatcherProvider) ResourceMatchers() []importmatcher.Matcher {
	return s.matchers
}

func TestMarkMatchedWith(t *testing.T) {
	t.Parallel()

	local := resources.NewResource("checkout", "category", resources.ResourceData{"name": "Checkout"}, []string{})
	localGraph := resources.NewGraph()
	localGraph.AddResource(local)

	// Any category remote matches the single local category.
	matcher := importmatcher.Matcher{
		ResourceType: "category",
		Match: func(_ importmatcher.Scope, _ *resources.RemoteResource) *resources.Resource {
			return local
		},
	}
	stub := stubMatcherProvider{matchers: []importmatcher.Matcher{matcher}}

	importableWith := func(rs ...*resources.RemoteResource) *resources.RemoteResources {
		collection := resources.NewRemoteResources()
		m := make(map[string]*resources.RemoteResource, len(rs))
		for _, r := range rs {
			m[r.ID] = r
		}
		collection.Set("category", m)
		return collection
	}

	t.Run("single match succeeds", func(t *testing.T) {
		t.Parallel()

		importable := importableWith(&resources.RemoteResource{ID: "cat_a", Reference: "#category:checkout-1"})

		err := markMatchedWith(stub, resources.NewGraph(), localGraph, importable)

		assert.NoError(t, err)
	})

	t.Run("multiple remotes matching one local fail fast", func(t *testing.T) {
		t.Parallel()

		importable := importableWith(
			&resources.RemoteResource{ID: "cat_a", Reference: "#category:checkout-1"},
			&resources.RemoteResource{ID: "cat_b", Reference: "#category:checkout-2"},
		)

		err := markMatchedWith(stub, resources.NewGraph(), localGraph, importable)

		require.ErrorIs(t, err, ErrAmbiguousMatch)
	})

	t.Run("no matcher provider is a no-op", func(t *testing.T) {
		t.Parallel()

		importable := importableWith(&resources.RemoteResource{ID: "cat_a", Reference: "#category:checkout-1"})

		err := markMatchedWith(stubMatcherProvider{}, resources.NewGraph(), localGraph, importable)

		assert.NoError(t, err)
	})
}
