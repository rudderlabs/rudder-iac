package common_test

import (
	"encoding/json"
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

func TestPropertiesMultipleSourceTypes(t *testing.T) {
	t.Parallel()

	props := common.Properties([]string{"web", "reactNative"})
	require.Len(t, props, 2)
}

func TestSchemaFragment(t *testing.T) {
	t.Parallel()

	fragment := common.SchemaFragment([]string{"web", "reactNative"})

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(fragment, &parsed))

	consentManagement, ok := parsed["consent_management"].(map[string]any)
	require.True(t, ok)

	properties, ok := consentManagement["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "web")
	assert.Contains(t, properties, "react_native")
}

func TestMergeSchemas(t *testing.T) {
	t.Parallel()

	base := json.RawMessage(`{
		"type": "object",
		"properties": {
			"webhook_url": { "type": "string" }
		}
	}`)
	fragment := common.SchemaFragment([]string{"web"})

	merged, err := common.MergeSchemas(base, fragment)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(merged, &parsed))

	properties := parsed["properties"].(map[string]any)
	assert.Contains(t, properties, "webhook_url")
	assert.Contains(t, properties, "consent_management")
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
