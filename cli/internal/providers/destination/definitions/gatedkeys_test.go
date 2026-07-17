package definitions

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func TestGatedKeyPaths(t *testing.T) {
	t.Parallel()

	def := GA4TestDefinition()
	def.Properties = []converter.ConfigProperty{
		converter.Simple("apiSecret", "api_secret"),
		converter.Gated(converter.Simple("measurementId", "measurement_id"), "web"),
		converter.Gated(converter.Simple("debugMode", "debug_mode"), "web", "android"),
	}

	registered, err := newRegisteredDefinition(def)
	require.NoError(t, err)

	assert.Equal(t, map[string][]string{
		"measurement_id": {"web"},
		"debug_mode":     {"web", "android"},
	}, registered.GatedKeyPaths())
}

func TestGatedKeyPathsReturnsCopy(t *testing.T) {
	t.Parallel()

	def := GA4TestDefinition()
	def.Properties = []converter.ConfigProperty{
		converter.Gated(converter.Simple("measurementId", "measurement_id"), "web"),
	}

	registered, err := newRegisteredDefinition(def)
	require.NoError(t, err)

	mutated := registered.GatedKeyPaths()
	mutated["measurement_id"][0] = "mutated"
	delete(mutated, "measurement_id")

	assert.Equal(t, map[string][]string{"measurement_id": {"web"}}, registered.GatedKeyPaths())
}

func TestGatedKeyPathsEmptyWithoutGatedProperties(t *testing.T) {
	t.Parallel()

	registered, err := newRegisteredDefinition(GA4TestDefinition())
	require.NoError(t, err)
	assert.Empty(t, registered.GatedKeyPaths())
}

func TestGatedKeyPathsNestedKey(t *testing.T) {
	t.Parallel()

	def := GA4TestDefinition()
	def.Properties = []converter.ConfigProperty{
		converter.Gated(converter.Simple("connectionMode.web", "connection_mode.web"), "web"),
	}

	registered, err := newRegisteredDefinition(def)
	require.NoError(t, err)

	assert.Equal(t, map[string][]string{
		"connection_mode/web": {"web"},
	}, registered.GatedKeyPaths())
}

func TestGatedKeyPathsErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		properties []converter.ConfigProperty
		errMessage string
	}{
		{
			name: "unsupported source type",
			properties: []converter.ConfigProperty{
				converter.Gated(converter.Simple("measurementId", "measurement_id"), "ios"),
			},
			errMessage: `config key "measurement_id" gated on unsupported source type "ios"`,
		},
		{
			name: "key not on config model",
			properties: []converter.ConfigProperty{
				converter.Gated(converter.Simple("unknownKey", "unknown_key"), "web"),
			},
			errMessage: `gated config key "unknown_key" does not resolve to a config model field`,
		},
		{
			name: "gated discriminator has no local key",
			properties: []converter.ConfigProperty{
				converter.Gated(converter.Discriminator("apiKey", converter.DiscriminatorValues{"k": "v"}), "web"),
			},
			errMessage: "gated config property has no local key",
		},
		{
			name: "duplicate gated key",
			properties: []converter.ConfigProperty{
				converter.Gated(converter.Simple("measurementId", "measurement_id"), "web"),
				converter.Gated(converter.Simple("measurementId", "measurement_id"), "android"),
			},
			errMessage: `duplicate gated config key "measurement_id"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			def := GA4TestDefinition()
			def.Properties = tc.properties

			_, err := newRegisteredDefinition(def)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMessage)
		})
	}
}

func TestConfigStructHasKeyPathThroughCollections(t *testing.T) {
	t.Parallel()

	type header struct {
		From string `mapstructure:"from"`
		To   string `mapstructure:"to"`
	}
	type config struct {
		Headers []header          `mapstructure:"headers"`
		Labels  map[string]header `mapstructure:"labels"`
	}

	typ := reflect.TypeOf(config{})

	assert.True(t, configStructHasKeyPath(typ, "headers.to"))
	assert.True(t, configStructHasKeyPath(typ, "labels.from"))
	assert.False(t, configStructHasKeyPath(typ, "headers.missing"))
	assert.False(t, configStructHasKeyPath(typ, "missing"))
}
