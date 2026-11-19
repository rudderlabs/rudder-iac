package connection

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateDestinationConfig_WarehouseEntry pins when a warehouse source may
// leave connection_mode out: only while cloud is the definition's single
// warehouse mode. A second mode makes the entry required again (fails closed),
// and the implied cloud mode still demands what cloud mode needs.
func TestValidateDestinationConfig_WarehouseEntry(t *testing.T) {
	t.Parallel()

	register := func(t *testing.T, typ string, warehouseModes []string, required map[string]map[string][]string) *definitions.RegisteredDefinition {
		t.Helper()
		registry := definitions.NewRegistry()
		require.NoError(t, registry.Register(&definitions.DestinationDefinition{
			Type: typ, DisplayName: "Test", Version: 1,
			NewConfig:              func() any { return &webhookTestConfig{} },
			SourceTypes:            []string{"web", common.SourceTypeWarehouse},
			ConnectionModes:        map[string][]string{"web": {"cloud"}, common.SourceTypeWarehouse: warehouseModes},
			ConnectionRequiredKeys: required,
		}))
		registered, err := registry.Get(typ, 1)
		require.NoError(t, err)
		return registered
	}

	cloudOnly := register(t, "cloud-only", []string{"cloud"}, map[string]map[string][]string{
		common.SourceTypeWarehouse: {"cloud": {"auth_token"}},
	})
	assert.Empty(t, ValidateDestinationConfig(cloudOnly, "/ref", "dest", common.SourceTypeWarehouse, map[string]any{"auth_token": "x"}))
	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/ref",
		Message:   "destination 'dest' config is missing fields required to connect a 'warehouse' source: auth_token",
	}}, ValidateDestinationConfig(cloudOnly, "/ref", "dest", common.SourceTypeWarehouse, map[string]any{}))
	assert.Empty(t, ValidateDestinationConfig(cloudOnly, "/ref", "dest", "web", map[string]any{}),
		"a cloud-capable event stream source may omit its mode")

	twoModes := register(t, "two-modes", []string{"cloud", "device"}, nil)
	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/ref",
		Message:   "destination 'dest' config has no 'connection_mode' entry for source type 'warehouse'",
	}}, ValidateDestinationConfig(twoModes, "/ref", "dest", common.SourceTypeWarehouse, map[string]any{}))
}
