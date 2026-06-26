package converter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func TestSimpleConfigProperty(t *testing.T) {
	t.Parallel()

	p := converter.Simple("a.b", "t.s")

	a, err := p.FromLocalFunc(`{ "p": true }`, `{ "t": { "s": "123" } }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true, "a": { "b": "123" } }`, a)

	s, err := p.ToLocalFunc(`{ "p": true }`, `{ "a": { "b": "123" } }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true, "t": { "s": "123" } }`, s)
}

func TestSimpleRoundTripViaMaps(t *testing.T) {
	t.Parallel()

	props := []converter.ConfigProperty{
		converter.Simple("webhookUrl", "webhook_url"),
		converter.Simple("debugMode", "debug_mode", converter.SkipZeroValue),
	}

	local := map[string]any{
		"webhook_url": "https://example.com",
		"debug_mode":  false,
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"webhookUrl": "https://example.com",
	}, api)

	back, err := converter.APIToLocal(props, api)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"webhook_url": "https://example.com",
	}, back)
}

func TestConditionalTrue(t *testing.T) {
	t.Parallel()

	p := converter.Conditional("a.b", "t.s", func(config string) bool {
		return true
	})

	a, err := p.FromLocalFunc(`{ "p": true }`, `{ "t": { "s": "123" } }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true, "a": { "b": "123" } }`, a)

	s, err := p.ToLocalFunc(`{ "p": true }`, `{ "a": { "b": "123" } }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true, "t": { "s": "123" } }`, s)
}

func TestSkipZeroValue(t *testing.T) {
	t.Parallel()

	assert.True(t, converter.SkipZeroValue(""))
	assert.True(t, converter.SkipZeroValue(0))
	assert.True(t, converter.SkipZeroValue(false))
	assert.True(t, converter.SkipZeroValue([]any{}))
	assert.False(t, converter.SkipZeroValue("123"))
	assert.False(t, converter.SkipZeroValue(123))
	assert.False(t, converter.SkipZeroValue(true))
	assert.False(t, converter.SkipZeroValue([]any{1, 2, 3}))
}

func TestConditionalFalse(t *testing.T) {
	t.Parallel()

	p := converter.Conditional("a.b", "t.s", func(config string) bool {
		return false
	})

	a, err := p.FromLocalFunc(`{ "p": true }`, `{ "t": { "s": "123" } }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true, "a": { "b": "123" } }`, a)

	s, err := p.ToLocalFunc(`{ "p": true }`, `{ "a": { "b": "123" } }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true }`, s)
}

func TestDiscriminator(t *testing.T) {
	t.Parallel()

	p := converter.Discriminator("f", converter.DiscriminatorValues{
		"foo": "FOO",
		"bar": "BAR",
	})

	a, err := p.FromLocalFunc(`{ "p": true }`, `{ "foo": true }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true, "f": "FOO" }`, a)

	a, err = p.FromLocalFunc(`{ "p": true }`, `{ "notfoo": true }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true }`, a)

	s, err := p.ToLocalFunc(`{ "p": true }`, `{ "f": "FOO" }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "p": true }`, s)
}

func TestEquals(t *testing.T) {
	t.Parallel()

	f := converter.Equals("a", "VALUE")
	assert.True(t, f(`{"a":"VALUE"}`))
	assert.False(t, f(`{"a":"NOT VALUE"}`))
	assert.False(t, f(`{"b":"VALUE"}`))
}

func TestArrayWithStrings(t *testing.T) {
	t.Parallel()

	p := converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist")

	a, err := p.FromLocalFunc(`{}`, `{ "event_filtering": { "whitelist": [ "a", "b" ] } }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"whitelistedEvents": [
			{ "eventName": "a" },
			{ "eventName": "b" }
		]
	}`, a)

	s, err := p.ToLocalFunc(`{}`, `{
		"whitelistedEvents": [
			{ "eventName": "a" },
			{ "eventName": "b" }
		]
	}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "event_filtering": { "whitelist": [ "a", "b" ] } }`, s)
}

func TestArrayWithObjects(t *testing.T) {
	t.Parallel()

	p := converter.ArrayWithObjects("eventChannelSettings", "event_channel_settings", map[string]any{
		"eventName":    "name",
		"eventChannel": "channel",
		"eventRegex":   "regex",
		"eventNestedValues": converter.APINestedObject{
			LocalKey:  "event_nested_values",
			NestedKey: "nestedKey",
		},
	})

	a, err := p.FromLocalFunc(`{}`, `{
		"event_channel_settings": [
			{ "name": "n1", "channel": "c1", "regex": "r1", "event_nested_values": [ "val1", "val2" ] },
			{ "name": "n2", "channel": "c2", "regex": "r2", "event_nested_values": [ "val3", "val4" ] }
		]
	}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"eventChannelSettings": [
			{ "eventName": "n1", "eventChannel": "c1", "eventRegex": "r1", "eventNestedValues": [ { "nestedKey": "val1" }, { "nestedKey": "val2" } ] },
			{ "eventName": "n2", "eventChannel": "c2", "eventRegex": "r2", "eventNestedValues": [ { "nestedKey": "val3" }, { "nestedKey": "val4" } ] }
		]
	}`, a)

	s, err := p.ToLocalFunc(`{}`, `{
		"eventChannelSettings": [
			{ "eventName": "n1", "eventChannel": "c1", "eventRegex": "r1", "extra": "e1", "eventNestedValues": [ { "nestedKey": "val1" }, { "nestedKey": "val2" } ] },
			{ "eventName": "n2", "eventChannel": "c2", "eventRegex": "r2", "extra": "e2", "eventNestedValues": [ { "nestedKey": "val3" }, { "nestedKey": "val4" } ] }
		]
	}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"event_channel_settings": [
			{ "name": "n1", "channel": "c1", "regex": "r1", "event_nested_values": [ "val1", "val2" ] },
			{ "name": "n2", "channel": "c2", "regex": "r2", "event_nested_values": [ "val3", "val4" ] }
		]
	}`, s)
}

func TestNestedSimplePath(t *testing.T) {
	t.Parallel()

	props := []converter.ConfigProperty{
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
	}

	local := map[string]any{
		"use_native_sdk": map[string]any{
			"web": true,
		},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"useNativeSDK": map[string]any{
			"web": true,
		},
	}, api)

	back, err := converter.APIToLocal(props, api)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}
