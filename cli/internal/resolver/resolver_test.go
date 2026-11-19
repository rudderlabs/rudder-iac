package resolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

func TestResolveToReferenceMissingCollectionResource(t *testing.T) {
	r := &resolver.ImportRefResolver{
		Importable: resources.NewRemoteResources(), Remote: resources.NewRemoteResources(), Graph: resources.NewGraph(),
	}
	ref, err := r.ResolveToReference("tracking-plan", "missing-tp")
	assert.ErrorIs(t, err, resolver.ErrResourceNotInCollection)
	assert.ErrorContains(t, err, "tracking-plan missing-tp")
	assert.Empty(t, ref)
}

func TestResolveToReferenceGraphFailureIsNotCollectionMissing(t *testing.T) {
	remote := resources.NewRemoteResources()
	remote.Set("tracking-plan", map[string]*resources.RemoteResource{
		"remote-tp": {ID: "remote-tp", ExternalID: "managed-tp"},
	})
	r := &resolver.ImportRefResolver{
		Importable: resources.NewRemoteResources(), Remote: remote, Graph: resources.NewGraph(),
	}
	ref, err := r.ResolveToReference("tracking-plan", "remote-tp")
	require.ErrorContains(t, err, "resource not present in resources graph")
	assert.NotErrorIs(t, err, resolver.ErrResourceNotInCollection)
	assert.Empty(t, ref)
}
