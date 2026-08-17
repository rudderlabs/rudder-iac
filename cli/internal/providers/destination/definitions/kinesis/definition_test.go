package kinesis_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/kinesis"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(kinesis.NewDefinition()))

	registered, err := registry.Get("kinesis", 1)
	require.NoError(t, err)

	assert.Equal(t, "kinesis", registered.Type)
	assert.Equal(t, "KINESIS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"access_key_id", "access_key"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "react_native", "flutter", "cordova", "cloud",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")
	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("KINESIS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestKinesisConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(kinesis.NewDefinition()))
	registered, err := registry.Get("kinesis", 1)
	require.NoError(t, err)

	minimalRoleConfig := func() map[string]any {
		return map[string]any{
			"region":          "us-east-1",
			"stream":          "rudder-cli-e2e-kinesis",
			"role_based_auth": true,
			"iam_role_arn":    "arn:aws:iam::123456789012:role/RudderKinesisAccess",
		}
	}

	minimalKeyConfig := func() map[string]any {
		return map[string]any{
			"region":          "us-east-1",
			"stream":          "rudder-cli-e2e-kinesis",
			"role_based_auth": false,
			"access_key_id":   "AKIAEXAMPLE",
			"access_key":      "secret-value",
		}
	}

	t.Run("missing region", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		delete(config, "region")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/region", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing stream", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		delete(config, "stream")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/stream", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("role_based_auth required", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		delete(config, "role_based_auth")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/role_based_auth", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("iam_role_arn required when role_based_auth true", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		delete(config, "iam_role_arn")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/iam_role_arn", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("access keys required when role_based_auth false", func(t *testing.T) {
		t.Parallel()
		config := minimalKeyConfig()
		delete(config, "access_key_id")
		delete(config, "access_key")

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 2)
		byPath := map[string]string{}
		for _, err := range errors {
			byPath[err.Path] = err.Message
		}
		assert.Equal(t, "'access_key_id' is required when 'role_based_auth' is false", byPath["/access_key_id"])
		assert.Equal(t, "'access_key' is required when 'role_based_auth' is false", byPath["/access_key"])
	})

	// schema.json's allOf branches only add requirements — neither forbids the
	// other mode's keys — so the CLI does not either. This also keeps a remote
	// config carrying a stale iam_role_arn importable, and matches S3, which has
	// the same four auth keys.
	t.Run("other mode keys are tolerated, matching schema.json and s3", func(t *testing.T) {
		t.Parallel()

		t.Run("access keys alongside role based auth", func(t *testing.T) {
			t.Parallel()
			config := minimalRoleConfig()
			config["access_key_id"] = "AKIAEXAMPLE"
			config["access_key"] = "secret-value"

			assert.Empty(t, registered.ValidateConfig(config))
		})

		t.Run("iam_role_arn alongside key based auth", func(t *testing.T) {
			t.Parallel()
			config := minimalKeyConfig()
			config["iam_role_arn"] = "arn:aws:iam::123456789012:role/RudderKinesisAccess"

			assert.Empty(t, registered.ValidateConfig(config))
		})
	})

	t.Run("valid role based auth with use_message_id", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		config["use_message_id"] = true

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("valid access key auth", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(minimalKeyConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid full config with consent", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		config["use_message_id"] = true
		config["consent_management"] = map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
		}

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("single line fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			field string
			value string
			path  string
		}{
			{name: "region over 100 characters", field: "region", value: strings.Repeat("a", 101), path: "/region"},
			{name: "stream with line break", field: "stream", value: "bad\nstream", path: "/stream"},
			{name: "iam role arn over 100 characters", field: "iam_role_arn", value: strings.Repeat("a", 101), path: "/iam_role_arn"},
			{name: "access key id over 100 characters", field: "access_key_id", value: strings.Repeat("a", 101), path: "/access_key_id"},
			{name: "access key with line break", field: "access_key", value: "bad\nsecret", path: "/access_key"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := minimalRoleConfig()
				if tc.field == "access_key_id" || tc.field == "access_key" {
					config = minimalKeyConfig()
				}
				config[tc.field] = tc.value

				errors := registered.ValidateConfig(config)

				require.Len(t, errors, 1)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("single line fields accept ui templates", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"region":          "{{ config.region || us-east-1 }}",
			"stream":          "{{ config.stream || rudder-cli-e2e-kinesis }}",
			"role_based_auth": true,
			"iam_role_arn":    "{{ config.iamRoleARN || arn:aws:iam::123456789012:role/RudderKinesisAccess }}",
		})

		assert.Empty(t, errors)
	})

	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		config["region"] = "env." + strings.Repeat("A", 101)

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/region", errors[0].Path)
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"region":          "us-east-1",
			"stream":          "rudder-cli-e2e-kinesis",
			"role_based_auth": true,
			"iam_role_arn":    "arn:aws:iam::123456789012:role/RudderKinesisAccess",
			"use_message_id":  true,
		})

		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		config["consent_management"] = map[string]any{
			"warehouse": []any{},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalRoleConfig()
		config["consent_management"] = map[string]any{
			"ios_swift": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestKinesisConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := kinesis.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "role based auth",
			LocalJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::123456789012:role/RudderKinesisAccess"
			}`,
			APIJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::123456789012:role/RudderKinesisAccess"
			}`,
		},
		{
			Name: "access key auth",
			LocalJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"role_based_auth": false,
				"access_key_id": "AKIAEXAMPLE",
				"access_key": "secret-value"
			}`,
			APIJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"roleBasedAuth": false,
				"accessKeyID": "AKIAEXAMPLE",
				"accessKey": "secret-value"
			}`,
		},
		{
			Name: "use message id",
			LocalJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::123456789012:role/RudderKinesisAccess",
				"use_message_id": true
			}`,
			APIJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::123456789012:role/RudderKinesisAccess",
				"useMessageId": true
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::123456789012:role/RudderKinesisAccess",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"region": "us-east-1",
				"stream": "rudder-cli-e2e-kinesis",
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::123456789012:role/RudderKinesisAccess",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

func TestKinesisRemoteReadExportRoundTrip(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(kinesis.NewDefinition()))
	registered, err := registry.Get("kinesis", 1)
	require.NoError(t, err)

	cases := []struct {
		name      string
		apiConfig map[string]any
		local     map[string]any
	}{
		{
			name: "role based auth",
			apiConfig: map[string]any{
				"region":        "us-east-1",
				"stream":        "rudder-cli-e2e-kinesis",
				"roleBasedAuth": true,
				"iamRoleARN":    "arn:aws:iam::123456789012:role/RudderKinesisAccess",
			},
			local: map[string]any{
				"region":          "us-east-1",
				"stream":          "rudder-cli-e2e-kinesis",
				"role_based_auth": true,
				"iam_role_arn":    "arn:aws:iam::123456789012:role/RudderKinesisAccess",
			},
		},
		{
			name: "access key auth",
			apiConfig: map[string]any{
				"region":        "us-east-1",
				"stream":        "rudder-cli-e2e-kinesis",
				"roleBasedAuth": false,
				"accessKeyID":   "AKIAEXAMPLE",
				"accessKey":     "secret-value",
			},
			local: map[string]any{
				"region":          "us-east-1",
				"stream":          "rudder-cli-e2e-kinesis",
				"role_based_auth": false,
				"access_key_id":   "AKIAEXAMPLE",
				"access_key":      "secret-value",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			local, err := registered.APIToLocal(tc.apiConfig)
			require.NoError(t, err)
			assert.Equal(t, tc.local, local)

			apiConfig, err := registered.LocalToAPI(local)
			require.NoError(t, err)
			assert.Equal(t, tc.apiConfig, apiConfig)
		})
	}
}
