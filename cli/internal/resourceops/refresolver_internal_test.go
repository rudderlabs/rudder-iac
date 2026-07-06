package resourceops

import (
	"context"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// refTestProvider is a full provider.Provider (via MockProvider) whose importable
// collection is configurable, so the resolver's unmanaged branch can be exercised.
type refTestProvider struct {
	*testutils.MockProvider
	importable *resources.RemoteResources
}

func (p *refTestProvider) LoadImportable(context.Context, namer.Namer) (*resources.RemoteResources, error) {
	return p.importable, nil
}

type fakeRefRouter struct{ provs map[string]provider.Provider }

func (r fakeRefRouter) ProviderForType(t string) (provider.Provider, error) {
	if p, ok := r.provs[t]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("no provider for %q", t)
}

func TestExportRefResolver(t *testing.T) {
	managed := resources.NewRemoteResources()
	managed.Set("tracking-plan", map[string]*resources.RemoteResource{
		"tp_managed": {ID: "tp_managed", ExternalID: "my-tp"},
	})
	importable := resources.NewRemoteResources()
	importable.Set("tracking-plan", map[string]*resources.RemoteResource{
		"tp_unmanaged": {ID: "tp_unmanaged", Reference: "#tracking-plan:suggested-tp"},
	})

	mp := testutils.NewMockProvider(nil, []string{"tracking-plan"})
	mp.LoadResourcesFromRemoteVal = managed
	router := fakeRefRouter{provs: map[string]provider.Provider{
		"tracking-plan": &refTestProvider{MockProvider: mp, importable: importable},
	}}

	r := newExportRefResolver(context.Background(), router)

	// Managed dependency → local external-id reference.
	ref, err := r.ResolveToReference("tracking-plan", "tp_managed")
	require.NoError(t, err)
	assert.Equal(t, "#tracking-plan:my-tp", ref)

	// Unmanaged (but existing) dependency → its adopt-ready suggested reference.
	ref, err = r.ResolveToReference("tracking-plan", "tp_unmanaged")
	require.NoError(t, err)
	assert.Equal(t, "#tracking-plan:suggested-tp", ref)

	// Missing dependency → degrade to the raw remote-id reference (no hard fail).
	ref, err = r.ResolveToReference("tracking-plan", "tp_ghost")
	require.NoError(t, err)
	assert.Equal(t, "#tracking-plan:tp_ghost", ref)
}

func TestExportRefResolver_NilRouterDegrades(t *testing.T) {
	r := newExportRefResolver(context.Background(), nil)
	ref, err := r.ResolveToReference("tracking-plan", "tp_x")
	require.NoError(t, err)
	assert.Equal(t, "#tracking-plan:tp_x", ref)
}
