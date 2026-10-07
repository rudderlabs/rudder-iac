package validator

import (
	"net"
	"testing"

	analytics "github.com/rudderlabs/analytics-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/ruddertyper"
)

// fakeClient records the messages it is given instead of sending them.
type fakeClient struct{ messages []analytics.Message }

func (c *fakeClient) Enqueue(msg analytics.Message) error {
	c.messages = append(c.messages, msg)
	return nil
}

func (c *fakeClient) Close() error { return nil }

// The golden's own event methods are called, so the test covers the whole
// generated call path: payload conversion, options and the send helper.
func TestGeneratedMethodsOwnTheirInputs(t *testing.T) {
	var (
		unicode = "a"
		props   = ruddertyper.TrackUserSignedUpProperties{
			ArrayOfAny:     []any{"x", []any{"y"}},
			MixedUnicode:   &unicode,
			ObjectProperty: map[string]any{"nested": map[string]any{"k": "v"}},
			PropertyOfAny:  []string{"z"},
		}
		open = map[string]any{"list": []any{"x"}}
		ctx  = analytics.Context{
			IP:     net.IP{127, 0, 0, 1},
			Extra:  map[string]any{"custom": map[string]any{"k": "v"}},
			Traits: analytics.Traits{"plan": []any{"pro"}},
		}
		integrations = analytics.Integrations{"Amplitude": map[string]any{"key": "v"}}
		client       = &fakeClient{}
		rt           = ruddertyper.New(client)
		identity     = ruddertyper.Identity{UserID: "user-123"}
		opts         = []ruddertyper.Option{nil, ruddertyper.WithAnalyticsContext(ctx), ruddertyper.WithIntegrations(integrations)}
	)

	require.NoError(t, rt.TrackUserSignedUp(identity, props, opts...))
	require.NoError(t, rt.TrackEmptyEventWithAdditionalProps(identity, open, opts...))

	unicode = "changed"
	props.ArrayOfAny[0] = "changed"
	props.ArrayOfAny[1].([]any)[0] = "changed"
	props.ObjectProperty["nested"].(map[string]any)["k"] = "changed"
	props.ObjectProperty["added"] = true
	props.PropertyOfAny.([]string)[0] = "changed"
	open["list"].([]any)[0] = "changed"
	open["added"] = true
	ctx.IP[0] = 10
	ctx.Extra["custom"].(map[string]any)["k"] = "changed"
	ctx.Traits["plan"].([]any)[0] = "changed"
	integrations["Amplitude"].(map[string]any)["key"] = "changed"
	integrations["All"] = false

	context := func() *analytics.Context {
		return &analytics.Context{
			IP: net.IP{127, 0, 0, 1},
			Extra: map[string]any{
				"custom": map[string]any{"k": "v"},
				"ruddertyper": map[string]any{
					"platform":            "go",
					"rudderCLIVersion":    "1.0.0",
					"trackingPlanId":      "plan_12345",
					"trackingPlanVersion": 13,
				},
			},
			Traits: analytics.Traits{"plan": []any{"pro"}},
		}
	}
	assert.Equal(t, []analytics.Message{
		analytics.Track{
			UserId: "user-123",
			Event:  "User Signed Up",
			Properties: analytics.Properties{
				"array_of_any":    []any{"x", []any{"y"}},
				"mixed_unicode":   "a",
				"object_property": map[string]any{"nested": map[string]any{"k": "v"}},
				"property_of_any": []any{"z"},
			},
			Context:      context(),
			Integrations: analytics.Integrations{"Amplitude": map[string]any{"key": "v"}},
		},
		analytics.Track{
			UserId:       "user-123",
			Event:        "Empty Event With Additional Props",
			Properties:   analytics.Properties{"list": []any{"x"}},
			Context:      context(),
			Integrations: analytics.Integrations{"Amplitude": map[string]any{"key": "v"}},
		},
	}, client.messages)
}
