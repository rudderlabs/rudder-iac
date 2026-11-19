package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImpliedConnectionMode pins cloud-capable event sources (including
// multi-mode ones) default to cloud, device-only ones do not, and a second
// warehouse mode disables the warehouse exception.
func TestImpliedConnectionMode(t *testing.T) {
	t.Parallel()

	registered := func(t *testing.T, warehouseModes []string) *RegisteredDefinition {
		t.Helper()
		def := GA4TestDefinition()
		def.SourceTypes = []string{"web", "android", "ios", "warehouse"}
		def.ConnectionModes = map[string][]string{
			"web": {"cloud", "device"}, "android": {"device"}, "ios": {"cloud"}, "warehouse": warehouseModes,
		}
		r, err := newRegisteredDefinition(def)
		require.NoError(t, err)
		return r
	}

	cloudOnly := registered(t, []string{"cloud"})
	assert.Equal(t, "cloud", cloudOnly.ImpliedConnectionMode("warehouse"))
	assert.Equal(t, "cloud", cloudOnly.ImpliedConnectionMode("web"))
	assert.Equal(t, "cloud", cloudOnly.ImpliedConnectionMode("ios"))
	assert.Empty(t, cloudOnly.ImpliedConnectionMode("android"))
	assert.Equal(t, "cloud", cloudOnly.ConnectMode("warehouse", map[string]any{}))
	assert.Equal(t, "cloud", cloudOnly.ConnectMode("warehouse", map[string]any{"connection_mode": map[string]any{"warehouse": "cloud"}}))
	assert.Equal(t, "cloud", cloudOnly.ConnectMode("web", map[string]any{}))
	assert.Equal(t, "device", cloudOnly.ConnectMode("web", map[string]any{"connection_mode": map[string]any{"web": "device"}}))
	assert.Empty(t, cloudOnly.ConnectMode("web", map[string]any{"connection_mode": map[string]any{"web": ""}}))
	assert.Empty(t, cloudOnly.ConnectMode("web", map[string]any{"connection_mode": map[string]any{"web": nil}}))

	twoModes := registered(t, []string{"cloud", "device"})
	assert.Equal(t, "", twoModes.ImpliedConnectionMode("warehouse"))
	assert.Equal(t, "device", twoModes.ConnectMode("warehouse", map[string]any{"connection_mode": map[string]any{"warehouse": "device"}}))

	noWarehouse := GA4TestDefinition()
	r, err := newRegisteredDefinition(noWarehouse)
	require.NoError(t, err)
	assert.Equal(t, "", r.ImpliedConnectionMode("warehouse"))
}
