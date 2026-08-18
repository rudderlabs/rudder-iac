package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestPropertiesSingleSourceType(t *testing.T) {
	t.Parallel()

	props := common.Properties([]string{"web"})
	require.Len(t, props, 1)

	local := map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider":            "oneTrust",
					"resolution_strategy": "and",
					"consents":            []any{"c1", "c2"},
				},
			},
		},
	}

	expectedAPI := map[string]any{
		"consentManagement": map[string]any{
			"web": []any{
				map[string]any{
					"provider":           "oneTrust",
					"resolutionStrategy": "and",
					"consents": []any{
						map[string]any{"consent": "c1"},
						map[string]any{"consent": "c2"},
					},
				},
			},
		},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, expectedAPI, api)

	back, err := converter.APIToLocal(props, expectedAPI)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

func TestPropertiesMapsLocalSourceTypeToAPI(t *testing.T) {
	t.Parallel()

	props := common.Properties([]string{"react_native"})
	local := map[string]any{
		"consent_management": map[string]any{
			"react_native": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
		},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"consentManagement": map[string]any{
			"reactnative": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{
						map[string]any{"consent": "analytics"},
					},
				},
			},
		},
	}, api)
}

func TestPropertiesMapsEveryKnownSourceTypeBothWays(t *testing.T) {
	t.Parallel()

	sourceMappings := common.LocalToAPISourceTypes()
	sourceTypes := make([]string, 0, len(sourceMappings))
	localSources := make(map[string]any, len(sourceMappings))
	apiSources := make(map[string]any, len(sourceMappings))
	for localSourceType, apiSourceType := range sourceMappings {
		sourceTypes = append(sourceTypes, localSourceType)
		localSources[localSourceType] = []any{map[string]any{"provider": "oneTrust"}}
		apiSources[apiSourceType] = []any{map[string]any{"provider": "oneTrust"}}
	}

	props := common.Properties(sourceTypes)
	require.Len(t, props, len(sourceMappings))

	local := map[string]any{"consent_management": localSources}
	expectedAPI := map[string]any{"consentManagement": apiSources}
	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, expectedAPI, api)

	back, err := converter.APIToLocal(props, expectedAPI)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

func TestOneTrustCookieCategoryProperties(t *testing.T) {
	t.Parallel()

	props := common.OneTrustCookieCategoryProperties([]string{"android_kotlin", "react_native"})
	require.Len(t, props, 2)

	local := map[string]any{
		"one_trust_cookie_categories": map[string]any{
			"android_kotlin": []any{map[string]any{"one_trust_cookie_category": "analytics"}},
			"react_native":   []any{map[string]any{"one_trust_cookie_category": "marketing"}},
		},
	}
	expectedAPI := map[string]any{
		"oneTrustCookieCategories": map[string]any{
			"androidKotlin": []any{map[string]any{"oneTrustCookieCategory": "analytics"}},
			"reactnative":   []any{map[string]any{"oneTrustCookieCategory": "marketing"}},
		},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, expectedAPI, api)

	back, err := converter.APIToLocal(props, expectedAPI)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

func TestKetchConsentPurposeProperties(t *testing.T) {
	t.Parallel()

	props := common.KetchConsentPurposeProperties([]string{"ios_swift", "web"})
	require.Len(t, props, 2)

	local := map[string]any{
		"ketch_consent_purposes": map[string]any{
			"ios_swift": []any{map[string]any{"purpose": "essential"}},
			"web":       []any{map[string]any{"purpose": "analytics"}},
		},
	}
	expectedAPI := map[string]any{
		"ketchConsentPurposes": map[string]any{
			"iosSwift": []any{map[string]any{"purpose": "essential"}},
			"web":      []any{map[string]any{"purpose": "analytics"}},
		},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, expectedAPI, api)

	back, err := converter.APIToLocal(props, expectedAPI)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

func TestAssertConversionHarness(t *testing.T) {
	t.Parallel()

	props := []converter.ConfigProperty{
		converter.Simple("webhookUrl", "webhook_url"),
	}

	testutil.AssertConversion(t, props, []testutil.ConversionCase{
		{
			Name:      "minimal",
			LocalJSON: `{"webhook_url":"https://example.com"}`,
			APIJSON:   `{"webhookUrl":"https://example.com"}`,
		},
	})
}
