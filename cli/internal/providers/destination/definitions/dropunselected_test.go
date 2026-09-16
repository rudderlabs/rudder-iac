package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

type testFilteringConfig struct {
	TrackingID     string              `mapstructure:"tracking_id" validate:"required"`
	EventFiltering *testEventFiltering `mapstructure:"event_filtering"`
}

type testEventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist"`
}

func testFilteringDefinition() *DestinationDefinition {
	return &DestinationDefinition{
		Type:    "FILTERTEST",
		Version: 1,
		Properties: []converter.ConfigProperty{
			converter.Simple("trackingID", "tracking_id"),
			converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
			converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
			converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
				"event_filtering.whitelist": "whitelistedEvents",
				"event_filtering.blacklist": "blacklistedEvents",
			}, converter.DropUnselected()),
		},
		NewConfig:       func() any { return &testFilteringConfig{} },
		SourceTypes:     []string{"web"},
		ConnectionModes: map[string][]string{"web": {"cloud"}},
	}
}

func events(names ...string) []any {
	out := []any{}
	for _, name := range names {
		out = append(out, map[string]any{"eventName": name})
	}
	return out
}

// The webapp keeps both lists and only switches the discriminator, so a
// destination that changed filtering mode — or cleared the unused list —
// converts to a spec declaring mutually exclusive keys together, which fails
// the definition's own validation and leaves the user to delete one by hand.
// A group declared with DropUnselected imports valid instead.
func TestAPIToLocalDropUnselectedImportsValid(t *testing.T) {
	t.Parallel()

	registered, err := newRegisteredDefinition(testFilteringDefinition())
	require.NoError(t, err)

	tests := []struct {
		name string
		api  map[string]any
		want any
	}{
		{
			name: "unselected list cleared by the webapp",
			api: map[string]any{
				"eventFilteringOption": "whitelistedEvents",
				"whitelistedEvents":    events("Order Completed"),
				"blacklistedEvents":    events(""),
			},
			want: map[string]any{"whitelist": []any{"Order Completed"}},
		},
		{
			name: "unselected list left over from a mode switch",
			api: map[string]any{
				"eventFilteringOption": "whitelistedEvents",
				"whitelistedEvents":    events("A"),
				"blacklistedEvents":    events("B"),
			},
			want: map[string]any{"whitelist": []any{"A"}},
		},
		{
			name: "selected empty list survives, it is the setting",
			api: map[string]any{
				"eventFilteringOption": "whitelistedEvents",
				"whitelistedEvents":    events(""),
				"blacklistedEvents":    events("B"),
			},
			want: map[string]any{"whitelist": []any{""}},
		},
		{
			name: "filtering disabled drops every member",
			api: map[string]any{
				"eventFilteringOption": "disable",
				"whitelistedEvents":    events("A"),
				"blacklistedEvents":    events("B"),
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := map[string]any{"trackingID": "UA-1"}
			for key, value := range tt.api {
				api[key] = value
			}

			local, err := registered.APIToLocal(api)
			require.NoError(t, err)

			if tt.want == nil {
				assert.NotContains(t, local, "event_filtering")
			} else {
				assert.Equal(t, tt.want, local["event_filtering"])
			}
			assert.Empty(t, registered.ValidateConfig(local), "the imported spec must pass its own validation")
		})
	}
}
