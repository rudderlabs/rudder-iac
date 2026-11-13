package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type defaultsTestConfig struct {
	APISecret     string `mapstructure:"api_secret" validate:"required"`
	MeasurementID string `mapstructure:"measurement_id"`
	Mode          string `mapstructure:"mode" default:"cloud"`
	DebugMode     *bool  `mapstructure:"debug_mode" default:"false"`
	SendUserID    bool   `mapstructure:"send_user_id" default:"true"`
	BatchSize     int    `mapstructure:"batch_size" default:"100"`
}

type nestedDefaultsTestConfig struct {
	APISecret  string `mapstructure:"api_secret" validate:"required"`
	SDKVersion *struct {
		Web *int `mapstructure:"web" default:"2"`
	} `mapstructure:"sdk_version"`
}

func registeredWithConfig(t *testing.T, newConfig func() any) *RegisteredDefinition {
	t.Helper()

	def := GA4TestDefinition()
	def.NewConfig = newConfig
	// These stubs replace GA4's config surface; scope its required and secret
	// keys to match.
	def.ConnectionRequiredKeys = nil
	def.SecretKeys = nil

	registered, err := newRegisteredDefinition(def)
	require.NoError(t, err)
	return registered
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultsTestConfig{} })

	assert.Equal(t, map[string]any{
		"mode":         "cloud",
		"debug_mode":   false,
		"send_user_id": true,
		"batch_size":   float64(100),
	}, registered.ConfigDefaults())
}

func TestConfigDefaultsReturnsCopy(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultsTestConfig{} })

	mutated := registered.ConfigDefaults()
	mutated["mode"] = "mutated"
	delete(mutated, "debug_mode")

	assert.Equal(t, map[string]any{
		"mode":         "cloud",
		"debug_mode":   false,
		"send_user_id": true,
		"batch_size":   float64(100),
	}, registered.ConfigDefaults())
}

func TestConfigDefaultsReturnsDeepCopy(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &nestedDefaultsTestConfig{} })

	mutated := registered.ConfigDefaults()
	mutated["sdk_version"].(map[string]any)["web"] = float64(1)

	assert.Equal(t, map[string]any{
		"sdk_version": map[string]any{"web": float64(2)},
	}, registered.ConfigDefaults())
}

func TestApplyDefaultsFillsOmittedKeys(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultsTestConfig{} })

	enriched := registered.ApplyDefaults(map[string]any{"api_secret": "secret"})

	assert.Equal(t, map[string]any{
		"api_secret":   "secret",
		"mode":         "cloud",
		"debug_mode":   false,
		"send_user_id": true,
		"batch_size":   float64(100),
	}, enriched)
}

func TestApplyDefaultsKeepsExplicitValues(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultsTestConfig{} })

	// A zero value is still an explicit value.
	enriched := registered.ApplyDefaults(map[string]any{
		"api_secret":   "secret",
		"mode":         "device",
		"send_user_id": false,
		"debug_mode":   false,
	})

	assert.Equal(t, map[string]any{
		"api_secret":   "secret",
		"mode":         "device",
		"send_user_id": false,
		"debug_mode":   false,
		"batch_size":   float64(100), // untouched keys still take their default
	}, enriched)
}

func TestApplyDefaultsDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultsTestConfig{} })

	config := map[string]any{"api_secret": "secret"}
	registered.ApplyDefaults(config)

	assert.Equal(t, map[string]any{"api_secret": "secret"}, config)
}

func TestApplyDefaultsLeavesAbsentNestedBlockAbsent(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &nestedDefaultsTestConfig{} })

	assert.Equal(t,
		map[string]any{"api_secret": "secret"},
		registered.ApplyDefaults(map[string]any{"api_secret": "secret"}),
	)
}

func TestApplyDefaultsKeepsExplicitNestedValue(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &nestedDefaultsTestConfig{} })

	assert.Equal(t, map[string]any{
		"api_secret":  "secret",
		"sdk_version": map[string]any{"web": 1},
	}, registered.ApplyDefaults(map[string]any{
		"api_secret":  "secret",
		"sdk_version": map[string]any{"web": 1},
	}))
}

func TestApplyDefaultsFillsPresentNestedBlock(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &nestedDefaultsTestConfig{} })

	assert.Equal(t, map[string]any{
		"api_secret":  "secret",
		"sdk_version": map[string]any{"web": float64(2)},
	}, registered.ApplyDefaults(map[string]any{
		"api_secret":  "secret",
		"sdk_version": map[string]any{},
	}))
}

func TestApplyDefaultsWithoutDeclaredDefaults(t *testing.T) {
	t.Parallel()

	registered, err := newRegisteredDefinition(GA4TestDefinition())
	require.NoError(t, err)

	assert.Empty(t, registered.ConfigDefaults())
	assert.Equal(t,
		map[string]any{"api_secret": "secret"},
		registered.ApplyDefaults(map[string]any{"api_secret": "secret"}),
	)
}

func TestRegisterRejectsInvalidDefaults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		newConfig func() any
		wantErr   string
	}{
		{
			name: "default on a required field",
			newConfig: func() any {
				return &struct {
					Mode string `mapstructure:"mode" validate:"required" default:"cloud"`
				}{}
			},
			wantErr: `config key "mode" is required and cannot declare a default`,
		},
		{
			name: "unparseable bool",
			newConfig: func() any {
				return &struct {
					DebugMode *bool `mapstructure:"debug_mode" default:"yes-please"`
				}{}
			},
			wantErr: `config key "debug_mode": invalid bool default "yes-please"`,
		},
		{
			name: "unparseable number",
			newConfig: func() any {
				return &struct {
					BatchSize int `mapstructure:"batch_size" default:"many"`
				}{}
			},
			wantErr: `config key "batch_size": invalid integer default "many"`,
		},
		{
			name: "fractional default on an integer field",
			newConfig: func() any {
				return &struct {
					BatchSize int `mapstructure:"batch_size" default:"1.5"`
				}{}
			},
			wantErr: `config key "batch_size": invalid integer default "1.5"`,
		},
		{
			name: "invalid default on a nested field",
			newConfig: func() any {
				return &struct {
					SDKVersion *struct {
						Web int `mapstructure:"web" default:"latest"`
					} `mapstructure:"sdk_version"`
				}{}
			},
			wantErr: `config key "sdk_version": config key "web": invalid integer default "latest"`,
		},
		{
			name: "default on a required nested field",
			newConfig: func() any {
				return &struct {
					SDKVersion *struct {
						Web int `mapstructure:"web" validate:"required" default:"2"`
					} `mapstructure:"sdk_version"`
				}{}
			},
			wantErr: `config key "sdk_version": config key "web" is required and cannot declare a default`,
		},
		{
			// Unimplemented: no upstream schema defaults these types.
			name: "unsigned field kind",
			newConfig: func() any {
				return &struct {
					RetryLimit uint `mapstructure:"retry_limit" default:"3"`
				}{}
			},
			wantErr: `config key "retry_limit": unsupported kind uint for a default tag`,
		},
		{
			name: "float field kind",
			newConfig: func() any {
				return &struct {
					Ratio float64 `mapstructure:"ratio" default:"0.5"`
				}{}
			},
			wantErr: `config key "ratio": unsupported kind float64 for a default tag`,
		},
		{
			name: "unsupported field kind",
			newConfig: func() any {
				return &struct {
					Limits map[string]string `mapstructure:"limits" default:"a"`
				}{}
			},
			wantErr: `config key "limits": unsupported kind map for a default tag`,
		},
		{
			// A list default carries strings only; anything else would have to
			// guess at an element encoding.
			name: "list of non-strings",
			newConfig: func() any {
				return &struct {
					Sizes []int `mapstructure:"sizes" default:"1,2"`
				}{}
			},
			wantErr: `config key "sizes": unsupported list element kind int for a default tag`,
		},
		{
			name: "list default with an empty element",
			newConfig: func() any {
				return &struct {
					Events []string `mapstructure:"events" default:"a,,b"`
				}{}
			},
			wantErr: `config key "events": invalid list default "a,,b": empty element`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			def := GA4TestDefinition()
			def.NewConfig = tc.newConfig
			// These stubs replace GA4's config surface; scope its required keys to match.
			def.ConnectionRequiredKeys = nil

			_, err := newRegisteredDefinition(def)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// A list default is kept as []any so it compares equal to the same JSON array
// decoded from an API response, which is what lets a spec that omits the key
// converge instead of diffing on every apply.
func TestStringListDefaultApplies(t *testing.T) {
	t.Parallel()

	def := GA4TestDefinition()
	def.ConnectionRequiredKeys = nil
	def.SecretKeys = nil
	def.NewConfig = func() any {
		return &struct {
			Schema []string `mapstructure:"schema" default:"EMAIL, PHONE"`
		}{}
	}

	registered, err := newRegisteredDefinition(def)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"schema": []any{"EMAIL", "PHONE"}}, registered.ApplyDefaults(map[string]any{}))
	assert.Equal(t,
		[]any{"MADID"},
		registered.ApplyDefaults(map[string]any{"schema": []any{"MADID"}})["schema"],
	)
}

func TestRegisterAllowsDefaultOnConditionallyRequiredField(t *testing.T) {
	t.Parallel()

	// required_if leaves the field optional, so a default still applies.
	registered := registeredWithConfig(t, func() any {
		return &struct {
			ClientType string `mapstructure:"client_type"`
			Mode       string `mapstructure:"mode" validate:"required_if=ClientType gtag" default:"cloud"`
		}{}
	})

	assert.Equal(t, map[string]any{"mode": "cloud"}, registered.ConfigDefaults())
}

func TestValidateConfigDefaultSatisfiesConditionalRequirement(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any {
		return &struct {
			UseIAM        *bool `mapstructure:"use_iam"`
			UseServerless *bool `mapstructure:"use_serverless" validate:"required_if=UseIAM true" default:"false"`
		}{}
	})

	assert.Empty(t, registered.ValidateConfig(map[string]any{"use_iam": true}))
}

func TestValidateConfigDefaultTriggersConditionalRequirement(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any {
		return &struct {
			RoleBasedAuth *bool  `mapstructure:"role_based_auth" default:"true"`
			IAMRoleARN    string `mapstructure:"iam_role_arn" validate:"required_if=RoleBasedAuth true"`
		}{}
	})

	assert.Equal(t, []ConfigError{{
		Path:    "/iam_role_arn",
		Message: "'iam_role_arn' is required when 'role_based_auth' is true",
	}}, registered.ValidateConfig(map[string]any{}))
}
