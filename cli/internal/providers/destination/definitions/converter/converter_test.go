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

// The discriminator is the only thing that says which of the mutually exclusive
// local keys is live, so it records the mapping for callers that need to tell a
// cleared unselected key from a deliberately empty selected one.
func TestLocalKeyForThreeStates(t *testing.T) {
	t.Parallel()

	p := converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
		"event_filtering.whitelist": "whitelistedEvents",
		"event_filtering.blacklist": "blacklistedEvents",
	})
	require.NotNil(t, p.Selector)

	// An absent discriminator says nothing about the members; one naming no
	// member positively says none is live.
	selected, present := p.Selector.LocalKeyFor(map[string]any{"eventFilteringOption": "blacklistedEvents"})
	assert.True(t, present)
	assert.Equal(t, "event_filtering.blacklist", selected)

	selected, present = p.Selector.LocalKeyFor(map[string]any{"eventFilteringOption": "disable"})
	assert.True(t, present, "the discriminator is present, it just names no member")
	assert.Empty(t, selected)

	selected, present = p.Selector.LocalKeyFor(map[string]any{})
	assert.False(t, present, "an absent discriminator is not a selection of none")
	assert.Empty(t, selected)
}

// Absent-discriminator and selected-empty behaviour rides on the real ga and adj
// definitions; this covers the shapes those two do not reach.
func TestAPIToLocalDropsUnselectedMembers(t *testing.T) {
	t.Parallel()

	arrays := []converter.ConfigProperty{
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
	}
	discriminator := converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
		"event_filtering.whitelist": "whitelistedEvents",
		"event_filtering.blacklist": "blacklistedEvents",
	})
	props := append(append([]converter.ConfigProperty{}, arrays...), discriminator)

	events := func(name string) []any { return []any{map[string]any{"eventName": name}} }
	apiConfig := map[string]any{
		"eventFilteringOption": "whitelistedEvents",
		"whitelistedEvents":    events("A"),
		"blacklistedEvents":    events("B"),
	}

	local, err := converter.APIToLocal(props, apiConfig)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"whitelist": []any{"A"}}, local["event_filtering"])

	// The drop runs in the APIToLocal driver, not in Discriminator's own
	// ToLocalFunc, so declaration order cannot change the result.
	local, err = converter.APIToLocal(append([]converter.ConfigProperty{discriminator}, arrays...), apiConfig)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"whitelist": []any{"A"}}, local["event_filtering"])

	// Naming no member means every member is unread; the whole group goes.
	local, err = converter.APIToLocal(props, map[string]any{
		"eventFilteringOption": "disable",
		"whitelistedEvents":    events("A"),
		"blacklistedEvents":    events("B"),
	})
	require.NoError(t, err)
	assert.NotContains(t, local, "event_filtering")
}

// An empty selected list is a setting, not an absence, so it has to survive
// both directions: the discriminator goes out on its own, and comes back as the
// empty list it stands for.
func TestAPIToLocalMaterializesSelectedKey(t *testing.T) {
	t.Parallel()

	props := []converter.ConfigProperty{
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}

	// The API stores the selector but no list for it.
	local, err := converter.APIToLocal(props, map[string]any{
		"eventFilteringOption": "whitelistedEvents",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"whitelist": []any{}}, local["event_filtering"])

	// A selector naming no member materializes nothing.
	local, err = converter.APIToLocal(props, map[string]any{"eventFilteringOption": "disable"})
	require.NoError(t, err)
	assert.NotContains(t, local, "event_filtering")

	// An absent selector materializes nothing either.
	local, err = converter.APIToLocal(props, map[string]any{})
	require.NoError(t, err)
	assert.NotContains(t, local, "event_filtering")

	// A list the API did carry is left as it is, not overwritten.
	local, err = converter.APIToLocal(props, map[string]any{
		"eventFilteringOption": "whitelistedEvents",
		"whitelistedEvents":    []any{map[string]any{"eventName": "A"}},
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"whitelist": []any{"A"}}, local["event_filtering"])

	// Outbound, an empty list still selects its branch.
	api, err := converter.LocalToAPI(props, map[string]any{
		"event_filtering": map[string]any{"whitelist": []any{}},
	})
	require.NoError(t, err)
	assert.Equal(t, "whitelistedEvents", api["eventFilteringOption"])

	// A null sibling selects nothing, so the choice stays deterministic even
	// though the discriminator iterates a map.
	for range 20 {
		api, err := converter.LocalToAPI(props, map[string]any{
			"event_filtering": map[string]any{"whitelist": []any{}, "blacklist": nil},
		})
		require.NoError(t, err)
		assert.Equal(t, "whitelistedEvents", api["eventFilteringOption"])
	}
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
		converter.Simple("connectionMode.web", "connection_mode.web"),
	}

	local := map[string]any{
		"connection_mode": map[string]any{
			"web": "cloud",
		},
	}

	api, err := converter.LocalToAPI(props, local)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"connectionMode": map[string]any{
			"web": "cloud",
		},
	}, api)

	back, err := converter.APIToLocal(props, api)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}
