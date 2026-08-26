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

type defaultJSONTestConfig struct {
	APISecret  string `mapstructure:"api_secret" validate:"required"`
	SDKVersion *struct {
		Web int `mapstructure:"web"`
	} `mapstructure:"sdk_version" default_json:"{\"web\":2}"`
}

func registeredWithConfig(t *testing.T, newConfig func() any) *RegisteredDefinition {
	t.Helper()

	def := GA4TestDefinition()
	def.NewConfig = newConfig
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

	registered := registeredWithConfig(t, func() any { return &defaultJSONTestConfig{} })

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

func TestApplyDefaultsFillsJSONObject(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultJSONTestConfig{} })

	assert.Equal(t, map[string]any{
		"api_secret":  "secret",
		"sdk_version": map[string]any{"web": float64(2)},
	}, registered.ApplyDefaults(map[string]any{"api_secret": "secret"}))
}

func TestApplyDefaultsKeepsExplicitJSONObject(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultJSONTestConfig{} })

	assert.Equal(t, map[string]any{
		"api_secret":  "secret",
		"sdk_version": map[string]any{"web": 1},
	}, registered.ApplyDefaults(map[string]any{
		"api_secret":  "secret",
		"sdk_version": map[string]any{"web": 1},
	}))
}

func TestApplyDefaultsMergesJSONObject(t *testing.T) {
	t.Parallel()

	registered := registeredWithConfig(t, func() any { return &defaultJSONTestConfig{} })

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
			name: "default and default_json on the same field",
			newConfig: func() any {
				return &struct {
					SDKVersion map[string]any `mapstructure:"sdk_version" default:"2" default_json:"{\"web\":2}"`
				}{}
			},
			wantErr: `config key "sdk_version" cannot declare both default and default_json`,
		},
		{
			name: "default_json on a required field",
			newConfig: func() any {
				return &struct {
					SDKVersion map[string]any `mapstructure:"sdk_version" validate:"required" default_json:"{\"web\":2}"`
				}{}
			},
			wantErr: `config key "sdk_version" is required and cannot declare a default_json`,
		},
		{
			name: "invalid default_json",
			newConfig: func() any {
				return &struct {
					SDKVersion map[string]any `mapstructure:"sdk_version" default_json:"{not-json}"`
				}{}
			},
			wantErr: `config key "sdk_version": invalid default_json "{not-json}"`,
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
					Events []string `mapstructure:"events" default:"a,b"`
				}{}
			},
			wantErr: `config key "events": unsupported kind slice for a default tag`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			def := GA4TestDefinition()
			def.NewConfig = tc.newConfig
			// These stubs drop GA4's consent/source-type fields, so scope the
			// definition down to what they model.
			def.SupportedSourcesValidation = nil

			_, err := newRegisteredDefinition(def)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
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
