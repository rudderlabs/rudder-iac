package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func TestConnectionModePropertiesSingleSourceType(t *testing.T) {
	t.Parallel()

	props := common.ConnectionModeProperties([]string{"web"})
	require.Len(t, props, 1)

	local := map[string]any{
		"connection_mode": map[string]any{"web": "device"},
	}
	expectedAPI := map[string]any{
		"connectionMode": map[string]any{"web": "device"},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, expectedAPI, api)

	back, err := converter.APIToLocal(props, expectedAPI)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

func TestConnectionModePropertiesMapsLocalSourceTypeToAPI(t *testing.T) {
	t.Parallel()

	props := common.ConnectionModeProperties([]string{"react_native"})
	local := map[string]any{
		"connection_mode": map[string]any{"react_native": "cloud"},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"connectionMode": map[string]any{"reactnative": "cloud"},
	}, api)
}

func TestConnectionModePropertiesMapsEveryKnownSourceTypeBothWays(t *testing.T) {
	t.Parallel()

	sourceMappings := common.LocalToAPISourceTypes()
	sourceTypes := make([]string, 0, len(sourceMappings))
	localSources := make(map[string]any, len(sourceMappings))
	apiSources := make(map[string]any, len(sourceMappings))
	for localSourceType, apiSourceType := range sourceMappings {
		sourceTypes = append(sourceTypes, localSourceType)
		localSources[localSourceType] = "cloud"
		apiSources[apiSourceType] = "cloud"
	}

	props := common.ConnectionModeProperties(sourceTypes)

	api, err := converter.LocalToAPI(props, map[string]any{"connection_mode": localSources})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"connectionMode": apiSources}, api)

	back, err := converter.APIToLocal(props, map[string]any{"connectionMode": apiSources})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"connection_mode": localSources}, back)
}

func TestConnectionModePropertiesEmptySourceTypes(t *testing.T) {
	t.Parallel()

	assert.Empty(t, common.ConnectionModeProperties(nil))
	assert.Empty(t, common.ConnectionModeProperties([]string{}))
}
