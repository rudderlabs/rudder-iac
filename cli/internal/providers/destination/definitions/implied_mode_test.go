package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImpliedConnectionMode pins that only a warehouse source on a definition
// whose single warehouse mode is cloud gets a mode without declaring one, and
// that a second warehouse mode switches that off rather than guessing.
func TestImpliedConnectionMode(t *testing.T) {
	t.Parallel()

	registered := func(t *testing.T, warehouseModes []string) *RegisteredDefinition {
		t.Helper()
		def := GA4TestDefinition()
		def.SourceTypes = []string{"web", "warehouse"}
		def.ConnectionModes = map[string][]string{"web": {"cloud"}, "warehouse": warehouseModes}
		r, err := newRegisteredDefinition(def)
		require.NoError(t, err)
		return r
	}

	cloudOnly := registered(t, []string{"cloud"})
	assert.Equal(t, "cloud", cloudOnly.ImpliedConnectionMode("warehouse"))
	assert.Equal(t, "", cloudOnly.ImpliedConnectionMode("web"), "an event stream source type never implies a mode, even with one offered")
	assert.Equal(t, "cloud", cloudOnly.ConnectMode("warehouse", map[string]any{}))
	assert.Equal(t, "cloud", cloudOnly.ConnectMode("warehouse", map[string]any{"connection_mode": map[string]any{"warehouse": "cloud"}}))
	assert.Equal(t, "", cloudOnly.ConnectMode("web", map[string]any{}))

	twoModes := registered(t, []string{"cloud", "device"})
	assert.Equal(t, "", twoModes.ImpliedConnectionMode("warehouse"))
	assert.Equal(t, "device", twoModes.ConnectMode("warehouse", map[string]any{"connection_mode": map[string]any{"warehouse": "device"}}))

	noWarehouse := GA4TestDefinition()
	r, err := newRegisteredDefinition(noWarehouse)
	require.NoError(t, err)
	assert.Equal(t, "", r.ImpliedConnectionMode("warehouse"))
}
