package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func TestNativeSDKSourceTypesSelectsDeviceRoutedModes(t *testing.T) {
	t.Parallel()

	sourceTypes := common.NativeSDKSourceTypes(map[string][]string{
		"web":            {"cloud", "device", "hybrid"},
		"android":        {"cloud", "device"},
		"android_kotlin": {"hybrid"},
		"cloud":          {"cloud"},
		"warehouse":      {"cloud"},
	})

	assert.Equal(t, []string{"android", "android_kotlin", "web"}, sourceTypes)
}

func TestNativeSDKSourceTypesWithoutDeviceRoutedModes(t *testing.T) {
	t.Parallel()

	assert.Nil(t, common.NativeSDKSourceTypes(map[string][]string{"cloud": {"cloud"}}))
	assert.Nil(t, common.NativeSDKSourceTypes(nil))
}

func TestNativeSDKPropertiesMapsLocalSourceTypesToAPI(t *testing.T) {
	t.Parallel()

	props := common.NativeSDKProperties(map[string][]string{
		"web":            {"cloud", "device", "hybrid"},
		"android_kotlin": {"cloud", "device"},
		"ios_swift":      {"cloud", "device"},
		"react_native":   {"cloud", "device"},
		"cloud":          {"cloud"},
	})
	require.Len(t, props, 4)

	local := map[string]any{
		"use_native_sdk": map[string]any{
			"web":            true,
			"android_kotlin": true,
			"ios_swift":      false,
			"react_native":   true,
		},
	}
	expectedAPI := map[string]any{
		"useNativeSDK": map[string]any{
			"web":           true,
			"androidKotlin": true,
			"iosSwift":      false,
			"reactnative":   true,
		},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, expectedAPI, api)

	back, err := converter.APIToLocal(props, expectedAPI)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

// Every source type must survive the round trip, so no definition can reach a
// spelling the canonical map does not cover.
func TestNativeSDKPropertiesMapsEveryKnownSourceTypeBothWays(t *testing.T) {
	t.Parallel()

	var (
		sourceMappings  = common.LocalToAPISourceTypes()
		connectionModes = make(map[string][]string, len(sourceMappings))
		localSources    = make(map[string]any, len(sourceMappings))
		apiSources      = make(map[string]any, len(sourceMappings))
	)
	for localSourceType, apiSourceType := range sourceMappings {
		connectionModes[localSourceType] = []string{"cloud", "device"}
		localSources[localSourceType] = true
		apiSources[apiSourceType] = true
	}

	props := common.NativeSDKProperties(connectionModes)
	require.Len(t, props, len(sourceMappings))

	local := map[string]any{"use_native_sdk": localSources}
	expectedAPI := map[string]any{"useNativeSDK": apiSources}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, expectedAPI, api)

	back, err := converter.APIToLocal(props, expectedAPI)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

func TestNativeSDKPropertiesWithoutDeviceRoutedModes(t *testing.T) {
	t.Parallel()

	assert.Nil(t, common.NativeSDKProperties(map[string][]string{"cloud": {"cloud"}}))
	assert.Nil(t, common.NativeSDKProperties(nil))
}
