package destination

import (
	"testing"

	"github.com/stretchr/testify/assert"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func TestSecretReferenceUniqueRuleMetadata(t *testing.T) {
	t.Parallel()

	rule := NewSecretReferenceUniqueRule(secretRuleRegistry(t))
	assert.Equal(t, SecretReferenceUniqueRuleID, rule.ID())
	assert.Equal(t, vrules.Warning, rule.Severity())
	assert.NotEmpty(t, rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(DestinationSpecKind), rule.AppliesTo())
	assert.Nil(t, rule.Validate(&vrules.ValidationContext{}))
	assert.Implements(t, (*vrules.SourceRule)(nil), rule)
}

func TestSecretReferenceUniqueRule(t *testing.T) {
	t.Parallel()

	const fix = "; give each secret key its own variable"

	cases := []struct {
		name    string
		sources map[string][]byte
		want    map[string][]vrules.ValidationResult
	}{
		{
			name: "distinct variables",
			sources: map[string][]byte{
				"braze.yaml": destinationSource("shop", "braze", "    data_center: US-01\n    android_api_key: '{{ .SHOP_ANDROID_API_KEY }}'\n"),
				"am.yaml":    destinationSource("shop-amplitude", "am", "    api_key: '{{ .SHOP_AMPLITUDE_API_KEY }}'\n    residency_server: standard\n"),
			},
		},
		{
			// Import gives both keys the same name: the id and the key are
			// joined by "_", and both contain "_" themselves.
			name: "names import would give two destinations",
			sources: map[string][]byte{
				"braze.yaml": destinationSource("shop", "braze", "    data_center: US-01\n    android_api_key: '{{ .SHOP_ANDROID_API_KEY }}'\n"),
				"am.yaml":    destinationSource("shop-android", "am", "    api_key: '{{ .SHOP_ANDROID_API_KEY }}'\n    residency_server: standard\n"),
			},
			want: map[string][]vrules.ValidationResult{
				"am.yaml": {{
					Reference: "/spec/config/api_key",
					Message:   `secret key "api_key" of destination "shop-android" shares variable SHOP_ANDROID_API_KEY with secret key "android_api_key" of destination "shop"` + fix,
				}},
				"braze.yaml": {{
					Reference: "/spec/config/android_api_key",
					Message:   `secret key "android_api_key" of destination "shop" shares variable SHOP_ANDROID_API_KEY with secret key "api_key" of destination "shop-android"` + fix,
				}},
			},
		},
		{
			name: "two keys of one destination and a list member elsewhere",
			sources: map[string][]byte{
				"am.yaml":      destinationSource("shop-amplitude", "am", "    api_key: '{{ .SHARED }}'\n    api_secret: '{{ .SHARED }}'\n    residency_server: standard\n"),
				"webhook.yaml": destinationSource("shop-webhook", "webhook", "    webhook_url: https://example.com/hook\n    headers:\n      - from: Authorization\n        to: '{{ .SHARED }}'\n"),
			},
			want: map[string][]vrules.ValidationResult{
				"am.yaml": {
					{
						Reference: "/spec/config/api_key",
						Message:   `secret key "api_key" of destination "shop-amplitude" shares variable SHARED with secret key "api_secret" of destination "shop-amplitude" and secret key "headers[0].to" of destination "shop-webhook"` + fix,
					},
					{
						Reference: "/spec/config/api_secret",
						Message:   `secret key "api_secret" of destination "shop-amplitude" shares variable SHARED with secret key "api_key" of destination "shop-amplitude" and secret key "headers[0].to" of destination "shop-webhook"` + fix,
					},
				},
				"webhook.yaml": {{
					Reference: "/spec/config/headers/0/to",
					Message:   `secret key "headers[0].to" of destination "shop-webhook" shares variable SHARED with secret key "api_key" of destination "shop-amplitude" and secret key "api_secret" of destination "shop-amplitude"` + fix,
				}},
			},
		},
		{
			// Ids that hold references, or none, read the same in every file,
			// so each destination is named by its file instead.
			name: "ids that do not tell destinations apart",
			sources: map[string][]byte{
				"am.yaml":  destinationSource("'{{ .DEST_ID }}'", "am", "    api_key: '{{ .SHARED }}'\n    residency_server: standard\n"),
				"am2.yaml": destinationSource(`""`, "am", "    api_key: '{{ .SHARED }}'\n    residency_server: standard\n"),
			},
			want: map[string][]vrules.ValidationResult{
				"am.yaml": {{
					Reference: "/spec/config/api_key",
					Message:   `secret key "api_key" of the destination in "am.yaml" shares variable SHARED with secret key "api_key" of the destination in "am2.yaml"` + fix,
				}},
				"am2.yaml": {{
					Reference: "/spec/config/api_key",
					Message:   `secret key "api_key" of the destination in "am2.yaml" shares variable SHARED with secret key "api_key" of the destination in "am.yaml"` + fix,
				}},
			},
		},
		{
			// Each warning names at most three other keys.
			name: "many keys sharing one variable",
			sources: map[string][]byte{
				"a.yaml": destinationSource("shop-a", "am", "    api_key: '{{ .SHARED }}'\n    api_secret: '{{ .SHARED }}'\n    residency_server: standard\n"),
				"b.yaml": destinationSource("shop-b", "am", "    api_key: '{{ .SHARED }}'\n    api_secret: '{{ .SHARED }}'\n    residency_server: standard\n"),
				"c.yaml": destinationSource("shop-c", "am", "    api_key: '{{ .SHARED }}'\n    residency_server: standard\n"),
			},
			want: map[string][]vrules.ValidationResult{
				"a.yaml": {
					{
						Reference: "/spec/config/api_key",
						Message:   `secret key "api_key" of destination "shop-a" shares variable SHARED with secret key "api_secret" of destination "shop-a", secret key "api_key" of destination "shop-b" and 2 other secret keys` + fix,
					},
					{
						Reference: "/spec/config/api_secret",
						Message:   `secret key "api_secret" of destination "shop-a" shares variable SHARED with secret key "api_key" of destination "shop-a", secret key "api_key" of destination "shop-b" and 2 other secret keys` + fix,
					},
				},
				"b.yaml": {
					{
						Reference: "/spec/config/api_key",
						Message:   `secret key "api_key" of destination "shop-b" shares variable SHARED with secret key "api_key" of destination "shop-a", secret key "api_secret" of destination "shop-a" and 2 other secret keys` + fix,
					},
					{
						Reference: "/spec/config/api_secret",
						Message:   `secret key "api_secret" of destination "shop-b" shares variable SHARED with secret key "api_key" of destination "shop-a", secret key "api_secret" of destination "shop-a" and 2 other secret keys` + fix,
					},
				},
				"c.yaml": {{
					Reference: "/spec/config/api_key",
					Message:   `secret key "api_key" of destination "shop-c" shares variable SHARED with secret key "api_key" of destination "shop-a", secret key "api_secret" of destination "shop-a" and 2 other secret keys` + fix,
				}},
			},
		},
		{
			// A reference with a default or with text around it already gets a
			// secret-reference warning, so it is not counted here.
			name: "only whole references count",
			sources: map[string][]byte{
				"am.yaml": destinationSource("shop-amplitude", "am", "    api_key: '{{ .SHARED }}'\n    api_secret: '{{ .SHARED | fallback }}'\n    residency_server: standard\n"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rule := NewSecretReferenceUniqueRule(secretRuleRegistry(t)).(vrules.SourceRule)
			results := rule.ValidateSources(sourceContexts(tc.sources))
			if tc.want == nil {
				assert.Empty(t, results)
				return
			}
			assert.Equal(t, tc.want, results)
		})
	}
}
