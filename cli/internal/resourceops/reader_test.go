package resourceops_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// fakeProvider satisfies provider.Provider by embedding EmptyProvider and
// overrides the two remote-load methods with configurable return values.
type fakeProvider struct {
	provider.EmptyProvider
	remote     *resources.RemoteResources
	importable *resources.RemoteResources
}

func (f *fakeProvider) SupportedKinds() []string  { return nil }
func (f *fakeProvider) SupportedTypes() []string   { return nil }
func (f *fakeProvider) LoadResourcesFromRemote(_ context.Context) (*resources.RemoteResources, error) {
	return f.remote, nil
}
func (f *fakeProvider) LoadImportable(_ context.Context, _ namer.Namer) (*resources.RemoteResources, error) {
	return f.importable, nil
}

func TestReader_List_MergesAndDedupes(t *testing.T) {
	managed := resources.NewRemoteResources()
	managed.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "a"},
	})
	importable := resources.NewRemoteResources()
	importable.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1"},  // dup of managed → managed wins
		"src_2": {ID: "src_2"}, // unmanaged only
	})
	prov := &fakeProvider{remote: managed, importable: importable}
	rows, err := resourceops.ListRows(context.Background(), prov, "event-stream-source", resourceops.ScopeAll)
	require.NoError(t, err)

	assert.ElementsMatch(t, []resourceops.Row{
		{ExternalID: "a", RemoteID: "src_1", Managed: true},
		{ExternalID: "", RemoteID: "src_2", Managed: false},
	}, rows)
}

func TestReader_List_ScopeManaged(t *testing.T) {
	managed := resources.NewRemoteResources()
	managed.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "a"},
	})
	importable := resources.NewRemoteResources()
	importable.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_2": {ID: "src_2"},
	})
	prov := &fakeProvider{remote: managed, importable: importable}
	rows, err := resourceops.ListRows(context.Background(), prov, "event-stream-source", resourceops.ScopeManaged)
	require.NoError(t, err)

	assert.ElementsMatch(t, []resourceops.Row{
		{ExternalID: "a", RemoteID: "src_1", Managed: true},
	}, rows)
}

func TestReader_List_ScopeUnmanaged(t *testing.T) {
	managed := resources.NewRemoteResources()
	managed.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "a"},
	})
	importable := resources.NewRemoteResources()
	importable.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1"},  // dup → managed wins, excluded from ScopeUnmanaged
		"src_2": {ID: "src_2"}, // unmanaged only
	})
	prov := &fakeProvider{remote: managed, importable: importable}
	rows, err := resourceops.ListRows(context.Background(), prov, "event-stream-source", resourceops.ScopeUnmanaged)
	require.NoError(t, err)

	assert.ElementsMatch(t, []resourceops.Row{
		{ExternalID: "", RemoteID: "src_2", Managed: false},
	}, rows)
}

func TestReader_List_NameExtractedFromData(t *testing.T) {
	managed := resources.NewRemoteResources()
	managed.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "a", Data: map[string]any{"name": "My Source"}},
	})
	prov := &fakeProvider{remote: managed, importable: resources.NewRemoteResources()}
	rows, err := resourceops.ListRows(context.Background(), prov, "event-stream-source", resourceops.ScopeAll)
	require.NoError(t, err)

	require.Len(t, rows, 1)
	assert.Equal(t, "My Source", rows[0].Name)
}

func TestReader_List_NoImportableSupport_Degraded(t *testing.T) {
	// MockProvider.LoadImportable always returns nil, nil — it implements
	// UnmanagedRemoteResourceLoader but returns nil collection; treat as empty.
	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "ext-a"},
	})
	mp.LoadResourcesFromRemoteVal = rc

	rows, err := resourceops.ListRows(context.Background(), mp, "event-stream-source", resourceops.ScopeAll)
	require.NoError(t, err)

	// Only managed row returned; no error because LoadImportable returned nil (empty).
	assert.ElementsMatch(t, []resourceops.Row{
		{ExternalID: "ext-a", RemoteID: "src_1", Managed: true},
	}, rows)

	// SupportsUnmanaged returns true for any provider implementing the interface,
	// even when it returns a nil collection (provider decides at runtime).
	assert.True(t, resourceops.SupportsUnmanaged(mp))
}

func TestReader_SupportsUnmanaged(t *testing.T) {
	// fakeProvider implements UnmanagedRemoteResourceLoader → true.
	prov := &fakeProvider{}
	assert.True(t, resourceops.SupportsUnmanaged(prov))

	// A provider NOT implementing the interface → false.
	// We use a struct that only embeds EmptyProvider but does NOT have LoadImportable.
	type bareProvider struct{ provider.EmptyProvider }
	// bareProvider does NOT satisfy UnmanagedRemoteResourceLoader because EmptyProvider
	// does not implement LoadImportable. Verify via the helper.
	assert.False(t, resourceops.SupportsUnmanaged(&bareProvider{}))
}

func TestReader_List_EmptyCollections(t *testing.T) {
	prov := &fakeProvider{
		remote:     resources.NewRemoteResources(),
		importable: resources.NewRemoteResources(),
	}
	rows, err := resourceops.ListRows(context.Background(), prov, "event-stream-source", resourceops.ScopeAll)
	require.NoError(t, err)
	assert.Empty(t, rows)
}
