package resourceops_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// fakeRouter routes a single known type to its provider; returns ErrUnsupportedType otherwise.
type fakeRouter struct {
	knownType string
	prov      provider.Provider
}

func (f *fakeRouter) ProviderForType(resourceType string) (provider.Provider, error) {
	if resourceType == f.knownType {
		return f.prov, nil
	}
	return nil, provider.ErrUnsupportedType
}

// mockExternalIDSetter embeds MockProvider and adds SetExternalID so it satisfies
// provider.ExternalIDSetter — used to test the positive capability-gate path.
type mockExternalIDSetter struct {
	*testutils.MockProvider
	setExternalIDErr error
}

func (m *mockExternalIDSetter) SetExternalID(_ context.Context, _, _, _ string) error {
	return m.setExternalIDErr
}

func TestResolver_FindRemote_ByExternalIDThenRemoteID(t *testing.T) {
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "my-source", Data: map[string]any{"name": "S"}},
	})

	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	mp.LoadResourcesFromRemoteVal = rc

	r := resourceops.New(&fakeRouter{knownType: "event-stream-source", prov: mp})

	// Find by external ID
	got, err := r.FindRemote(context.Background(), "event-stream-source", "my-source")
	require.NoError(t, err)
	assert.Equal(t, "src_1", got.ID)

	// Find by remote ID
	got2, err := r.FindRemote(context.Background(), "event-stream-source", "src_1")
	require.NoError(t, err)
	assert.Equal(t, "src_1", got2.ID)

	// Not found
	_, err = r.FindRemote(context.Background(), "event-stream-source", "ghost")
	assert.ErrorIs(t, err, resourceops.ErrResourceNotFound)
}

func TestResolver_FindRemote_UnknownType(t *testing.T) {
	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	r := resourceops.New(&fakeRouter{knownType: "event-stream-source", prov: mp})

	_, err := r.FindRemote(context.Background(), "unknown-type", "any-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, provider.ErrUnsupportedType)
}

func TestResolver_FindRemote_EmptyCollection(t *testing.T) {
	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	// LoadResourcesFromRemoteVal is nil — provider returns nil collection
	mp.LoadResourcesFromRemoteVal = resources.NewRemoteResources()

	r := resourceops.New(&fakeRouter{knownType: "event-stream-source", prov: mp})

	_, err := r.FindRemote(context.Background(), "event-stream-source", "any-id")
	assert.ErrorIs(t, err, resourceops.ErrResourceNotFound)
}

func TestResolver_ExternalIDSetterFor_NotSupported(t *testing.T) {
	// MockProvider does NOT implement ExternalIDSetter — gate should reject.
	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	r := resourceops.New(&fakeRouter{knownType: "event-stream-source", prov: mp})

	_, err := r.ExternalIDSetterFor("event-stream-source")
	require.Error(t, err)
	assert.ErrorIs(t, err, resourceops.ErrVerbNotSupported)
}

func TestResolver_ExternalIDSetterFor_Supported(t *testing.T) {
	// mockExternalIDSetter embeds MockProvider and adds SetExternalID — gate should pass.
	base := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	mp := &mockExternalIDSetter{MockProvider: base}
	r := resourceops.New(&fakeRouter{knownType: "event-stream-source", prov: mp})

	setter, err := r.ExternalIDSetterFor("event-stream-source")
	require.NoError(t, err)
	require.NotNil(t, setter)

	// Verify the returned setter is functional.
	err = setter.SetExternalID(context.Background(), "event-stream-source", "src_1", "my-source")
	assert.NoError(t, err)
}

func TestResolver_ExternalIDSetterFor_UnknownType(t *testing.T) {
	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	r := resourceops.New(&fakeRouter{knownType: "event-stream-source", prov: mp})

	_, err := r.ExternalIDSetterFor("unknown-type")
	require.Error(t, err)
	// Error comes from router, not ErrVerbNotSupported
	assert.ErrorIs(t, err, provider.ErrUnsupportedType)
}

func TestResolver_ProviderFor(t *testing.T) {
	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	r := resourceops.New(&fakeRouter{knownType: "event-stream-source", prov: mp})

	got, err := r.ProviderFor("event-stream-source")
	require.NoError(t, err)
	assert.Equal(t, mp, got)

	_, err = r.ProviderFor("unknown-type")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ValidateType tests
// ---------------------------------------------------------------------------

func TestValidateType_KnownType_ReturnsNil(t *testing.T) {
	t.Parallel()

	types := []string{"event-stream-source", "event-stream-destination"}
	require.NoError(t, resourceops.ValidateType(types, "event-stream-source"))
	require.NoError(t, resourceops.ValidateType(types, "event-stream-destination"))
}

func TestValidateType_UnknownType_ListsValidTypes(t *testing.T) {
	t.Parallel()

	types := []string{"event-stream-source", "event-stream-destination"}
	err := resourceops.ValidateType(types, "bogus-type")
	require.Error(t, err)

	msg := err.Error()
	assert.Contains(t, msg, "bogus-type", "error must name the offending type")
	assert.Contains(t, msg, "event-stream-source", "error must list valid types")
	assert.Contains(t, msg, "event-stream-destination", "error must list valid types")
}

func TestValidateType_EmptyList_ReturnsError(t *testing.T) {
	t.Parallel()

	err := resourceops.ValidateType([]string{}, "anything")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "anything")
}
