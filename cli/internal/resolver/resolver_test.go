package resolver_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportRefResolver_ResolveToReference(t *testing.T) {
	entityType := "event-stream-source"
	remoteID := "remote-123"
	externalID := "local-ext-id"
	metadataRef := "sources/event-stream-source.yaml"

	cases := []struct {
		name        string
		resolver    *resolver.ImportRefResolver
		wantRef     string
		wantErr     bool
		errContains string
	}{
		{
			name: "resolves reference from importable collection",
			resolver: &resolver.ImportRefResolver{
				Importable: resources.NewRemoteResources(),
				Remote:     resources.NewRemoteResources(),
				Graph:      resources.NewGraph(),
			},
			wantRef: "importable-ref",
		},
		{
			name: "resolves metadata ref from remote resource in graph",
			resolver: &resolver.ImportRefResolver{
				Importable: resources.NewRemoteResources(),
				Remote:     resources.NewRemoteResources(),
				Graph:      resources.NewGraph(),
			},
			wantRef: metadataRef,
		},
		{
			name: "returns error when remote resource is missing",
			resolver: &resolver.ImportRefResolver{
				Importable: resources.NewRemoteResources(),
				Remote:     resources.NewRemoteResources(),
				Graph:      resources.NewGraph(),
			},
			wantErr:     true,
			errContains: "resource not present in resources collection",
		},
		{
			name: "returns error when graph resource is missing",
			resolver: &resolver.ImportRefResolver{
				Importable: resources.NewRemoteResources(),
				Remote:     resources.NewRemoteResources(),
				Graph:      resources.NewGraph(),
			},
			wantErr:     true,
			errContains: "resource not present in resources graph",
		},
		{
			name: "returns error when graph resource has no file metadata",
			resolver: &resolver.ImportRefResolver{
				Importable: resources.NewRemoteResources(),
				Remote:     resources.NewRemoteResources(),
				Graph:      resources.NewGraph(),
			},
			wantErr:     true,
			errContains: "file metadata on the graph resource is not present",
		},
		{
			name: "returns error when graph resource metadata ref is empty",
			resolver: &resolver.ImportRefResolver{
				Importable: resources.NewRemoteResources(),
				Remote:     resources.NewRemoteResources(),
				Graph:      resources.NewGraph(),
			},
			wantErr:     true,
			errContains: "file metadata on the graph resource is not present",
		},
	}

	cases[0].resolver.Importable.Set(entityType, map[string]*resources.RemoteResource{
		remoteID: {
			ID:        remoteID,
			Reference: "importable-ref",
		},
	})

	cases[1].resolver.Remote.Set(entityType, map[string]*resources.RemoteResource{
		remoteID: {
			ID:         remoteID,
			ExternalID: externalID,
		},
	})
	graphResource := resources.NewResource(
		externalID,
		entityType,
		resources.ResourceData{"name": "test-source"},
		nil,
		resources.WithResourceFileMetadata(metadataRef),
	)
	cases[1].resolver.Graph.AddResource(graphResource)

	cases[3].resolver.Remote.Set(entityType, map[string]*resources.RemoteResource{
		remoteID: {
			ID:         remoteID,
			ExternalID: externalID,
		},
	})

	cases[4].resolver.Remote.Set(entityType, map[string]*resources.RemoteResource{
		remoteID: {
			ID:         remoteID,
			ExternalID: externalID,
		},
	})
	cases[4].resolver.Graph.AddResource(resources.NewResource(
		externalID,
		entityType,
		resources.ResourceData{"name": "test-source"},
		nil,
	))

	cases[5].resolver.Remote.Set(entityType, map[string]*resources.RemoteResource{
		remoteID: {
			ID:         remoteID,
			ExternalID: externalID,
		},
	})
	cases[5].resolver.Graph.AddResource(resources.NewResource(
		externalID,
		entityType,
		resources.ResourceData{"name": "test-source"},
		nil,
		resources.WithResourceFileMetadata(""),
	))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ref, err := tc.resolver.ResolveToReference(entityType, remoteID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantRef, ref)
		})
	}
}
