package resolver

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveToReference_PendingDeleteConflict(t *testing.T) {
	remote := resources.NewRemoteResources()
	remote.Set("category", map[string]*resources.RemoteResource{
		"cat_remote": {
			ID:         "cat_remote",
			ExternalID: "legacy-category",
		},
	})

	resolver := &ImportRefResolver{
		Remote:     remote,
		Graph:      resources.NewGraph(),
		Importable: resources.NewRemoteResources(),
	}

	_, err := resolver.ResolveToReference("category", "cat_remote")

	require.ErrorIs(t, err, ErrPendingDeleteConflict)
	assert.Contains(t, err.Error(), resources.URN("legacy-category", "category"))
	assert.Contains(t, err.Error(), "cat_remote")
}
