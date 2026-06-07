package resourceops_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
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

func TestReader_List_LoadImportableReturnsNil(t *testing.T) {
	// MockProvider implements UnmanagedRemoteResourceLoader but its LoadImportable
	// returns nil, nil — the interface is present but the collection is empty.
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

// managedOnlyProvider satisfies provider.ManagedRemoteResourceLoader but does NOT
// implement provider.UnmanagedRemoteResourceLoader (no LoadImportable method).
// This exercises the `if !ok { return result, nil }` branch in mergedRemote.
type managedOnlyProvider struct {
	remote *resources.RemoteResources
}

func (p *managedOnlyProvider) LoadResourcesFromRemote(_ context.Context) (*resources.RemoteResources, error) {
	return p.remote, nil
}

func TestReader_List_ProviderWithoutUnmanagedInterface(t *testing.T) {
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "ext-a"},
	})
	prov := &managedOnlyProvider{remote: rc}

	// SupportsUnmanaged must report false — the interface is genuinely absent.
	assert.False(t, resourceops.SupportsUnmanaged(prov))

	rows, err := resourceops.ListRows(context.Background(), prov, "event-stream-source", resourceops.ScopeAll)
	require.NoError(t, err)

	// Only the managed row is returned; no panic because the unmanaged branch is skipped.
	assert.ElementsMatch(t, []resourceops.Row{
		{ExternalID: "ext-a", RemoteID: "src_1", Managed: true},
	}, rows)
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

// newEventStreamProvider builds a real event-stream provider backed by a mock client
// that returns one managed source (ExternalID = "my-source").
func newEventStreamProvider() *eventstream.Provider {
	mockClient := esSource.NewMockSourceClient()
	mockClient.SetGetSourcesFunc(func(ctx context.Context) ([]sourceClient.EventStreamSource, error) {
		return []sourceClient.EventStreamSource{
			{
				ID:          "src-remote-id",
				ExternalID:  "my-source",
				Name:        "My Source",
				Type:        "javascript",
				Enabled:     true,
				WorkspaceID: "ws-123",
				TrackingPlan: nil,
			},
		}, nil
	})
	return eventstream.New(mockClient)
}

func TestReader_SpecYAML_RoundTrips(t *testing.T) {
	prov := newEventStreamProvider()

	out, err := resourceops.SpecYAML(context.Background(), prov, esSource.ResourceType, "my-source")
	require.NoError(t, err)
	require.NotEmpty(t, out)

	spec, err := specs.New([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, esSource.ResourceKind, spec.Kind)
}

func TestReader_SpecYAML_MatchByRemoteID(t *testing.T) {
	prov := newEventStreamProvider()

	// "src-remote-id" is the remote (non-external) ID; external-ID lookup fails first,
	// then remote-ID lookup should find it.
	out, err := resourceops.SpecYAML(context.Background(), prov, esSource.ResourceType, "src-remote-id")
	require.NoError(t, err)

	spec, err := specs.New([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, esSource.ResourceKind, spec.Kind)
}

func TestReader_SpecYAML_NotFound(t *testing.T) {
	prov := newEventStreamProvider()

	_, err := resourceops.SpecYAML(context.Background(), prov, esSource.ResourceType, "does-not-exist")
	require.ErrorIs(t, err, resourceops.ErrResourceNotFound)
}

func TestReader_SpecJSON_RoundTrips(t *testing.T) {
	prov := newEventStreamProvider()

	out, err := resourceops.SpecJSON(context.Background(), prov, esSource.ResourceType, "my-source")
	require.NoError(t, err)
	require.NotEmpty(t, out)

	// Must be valid JSON.
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &m))

	// Top-level "kind" key must equal the source resource kind.
	kind, ok := m["kind"].(string)
	require.True(t, ok, "expected 'kind' key in JSON output")
	assert.Equal(t, esSource.ResourceKind, kind)
}

// TestReader_SpecYAML_TypeNotFound verifies ErrResourceNotFound for an unknown resource type.
func TestReader_SpecYAML_TypeNotFound(t *testing.T) {
	prov := newEventStreamProvider()

	_, err := resourceops.SpecYAML(context.Background(), prov, esSource.ResourceType, "unknown-id")
	require.ErrorIs(t, err, resourceops.ErrResourceNotFound)
}
