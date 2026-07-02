package definitions

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type unknownKeysConsentItem struct {
	Provider string `mapstructure:"provider"`
}

type unknownKeysConsentBySource struct {
	Web []unknownKeysConsentItem `mapstructure:"web"`
}

type unknownKeysConfig struct {
	APISecret      string                     `mapstructure:"api_secret"`
	ConnectionMode testConnectionMode         `mapstructure:"connection_mode"`
	Consent        unknownKeysConsentBySource `mapstructure:"consent_management"`
}

func TestFindUnknownKeysTopLevel(t *testing.T) {
	t.Parallel()

	errors := findUnknownKeys(map[string]any{
		"api_secret":    "secret",
		"unknown_field": "value",
	}, reflect.TypeOf(testGA4Config{}), "")

	require.Len(t, errors, 1)
	assert.Equal(t, ConfigError{
		Path:    "/unknown_field",
		Message: `unknown config field "unknown_field"`,
	}, errors[0])
}

func TestFindUnknownKeysNestedStruct(t *testing.T) {
	t.Parallel()

	errors := findUnknownKeys(map[string]any{
		"api_secret": "secret",
		"connection_mode": map[string]any{
			"web":     "cloud",
			"unknown": "cloud",
		},
	}, reflect.TypeOf(testGA4Config{}), "")

	require.Len(t, errors, 1)
	assert.Equal(t, "/connection_mode/unknown", errors[0].Path)
	assert.Equal(t, `unknown config field "unknown"`, errors[0].Message)
}

func TestFindUnknownKeysMultipleLevels(t *testing.T) {
	t.Parallel()

	errors := findUnknownKeys(map[string]any{
		"api_secret":    "secret",
		"extra_top":     "x",
		"connection_mode": map[string]any{
			"web":       "cloud",
			"extra_src": "cloud",
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"provider": "oneTrust"},
			},
			"extra_source": []any{},
		},
	}, reflect.TypeOf(unknownKeysConfig{}), "")

	assertConfigErrors(t, errors,
		ConfigError{Path: "/extra_top", Message: `unknown config field "extra_top"`},
		ConfigError{Path: "/connection_mode/extra_src", Message: `unknown config field "extra_src"`},
		ConfigError{Path: "/consent_management/extra_source", Message: `unknown config field "extra_source"`},
	)
}

func TestFindUnknownKeysSliceItem(t *testing.T) {
	t.Parallel()

	errors := findUnknownKeys(map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"bogus":    "x",
				},
			},
		},
	}, reflect.TypeOf(unknownKeysConfig{}), "")

	require.Len(t, errors, 1)
	assert.Equal(t, "/consent_management/web/0/bogus", errors[0].Path)
	assert.Equal(t, `unknown config field "bogus"`, errors[0].Message)
}

func TestFindUnknownKeysValidConfig(t *testing.T) {
	t.Parallel()

	errors := findUnknownKeys(map[string]any{
		"api_secret":      "secret",
		"types_of_client": "gtag",
		"connection_mode": map[string]any{
			"web": "cloud",
		},
	}, reflect.TypeOf(testGA4Config{}), "")

	assert.Empty(t, errors)
}

func TestFindUnknownKeysWithBasePath(t *testing.T) {
	t.Parallel()

	errors := findUnknownKeys(map[string]any{
		"unknown_field": "value",
	}, reflect.TypeOf(testGA4Config{}), "/config")

	require.Len(t, errors, 1)
	assert.Equal(t, "/config/unknown_field", errors[0].Path)
}

func TestFindUnknownKeysNonMapInput(t *testing.T) {
	t.Parallel()

	assert.Nil(t, findUnknownKeys("not-a-map", reflect.TypeOf(testGA4Config{}), ""))
}

func TestFindUnknownKeysNonStructType(t *testing.T) {
	t.Parallel()

	assert.Nil(t, findUnknownKeys(map[string]any{"key": "value"}, reflect.TypeOf(""), ""))
}

func TestFindUnknownKeysLeafTypeMismatch(t *testing.T) {
	t.Parallel()

	// api_secret expects a string; a nested map is a decode error, not an unknown-key error.
	errors := findUnknownKeys(map[string]any{
		"api_secret": map[string]any{"nested": "value"},
	}, reflect.TypeOf(testGA4Config{}), "")

	assert.Empty(t, errors)
}

func TestJoinConfigPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/segment", joinConfigPath("", "segment"))
	assert.Equal(t, "/config/segment", joinConfigPath("/config", "segment"))
	assert.Equal(t, "/config/nested/0", joinConfigPath("/config/nested", "0"))
}

func TestMapstructureFieldTag(t *testing.T) {
	t.Parallel()

	type tagged struct {
		WithTag    string `mapstructure:"with_tag"`
		Ignored    string `mapstructure:"-"`
		NoTag      string
		EmptyTag   string `mapstructure:""`
		Squashed   string `mapstructure:"squashed,omitempty"`
		unexported string `mapstructure:"hidden"`
	}

	withTag, ok := mapstructureFieldTag(reflect.TypeOf(tagged{}).Field(0))
	assert.True(t, ok)
	assert.Equal(t, "with_tag", withTag)

	_, ok = mapstructureFieldTag(reflect.TypeOf(tagged{}).Field(1))
	assert.False(t, ok)

	_, ok = mapstructureFieldTag(reflect.TypeOf(tagged{}).Field(2))
	assert.False(t, ok)

	_, ok = mapstructureFieldTag(reflect.TypeOf(tagged{}).Field(3))
	assert.False(t, ok)

	squashed, ok := mapstructureFieldTag(reflect.TypeOf(tagged{}).Field(4))
	assert.True(t, ok)
	assert.Equal(t, "squashed", squashed)
}

func TestStructFieldsByMapstructureTag(t *testing.T) {
	t.Parallel()

	fields := structFieldsByMapstructureTag(reflect.TypeOf(testGA4Config{}))

	assert.Contains(t, fields, "api_secret")
	assert.Contains(t, fields, "connection_mode")
	assert.NotContains(t, fields, "APISecret")
}

func assertConfigErrors(t *testing.T, errors []ConfigError, expected ...ConfigError) {
	t.Helper()

	require.Len(t, errors, len(expected))
	assert.ElementsMatch(t, expected, errors)
}
