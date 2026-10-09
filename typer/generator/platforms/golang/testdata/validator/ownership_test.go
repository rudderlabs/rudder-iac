package validator

import (
	"fmt"
	"net"
	"sync"
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

// sendAndChange sends two events through client and changes everything it
// passed in as soon as each call returns.
func sendAndChange(t *testing.T, client analytics.Client) {
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
		rt           = ruddertyper.New(client)
		identity     = ruddertyper.Identity{UserID: "user-123"}
		opts         = []ruddertyper.Option{nil, ruddertyper.WithAnalyticsContext(ctx), ruddertyper.WithIntegrations(integrations)}
	)

	require.NoError(t, rt.TrackUserSignedUp(identity, props, opts...))
	unicode = "changed"
	props.ArrayOfAny[0] = "changed"
	props.ArrayOfAny[1].([]any)[0] = "changed"
	props.ObjectProperty["nested"].(map[string]any)["k"] = "changed"
	props.ObjectProperty["added"] = true
	props.PropertyOfAny.([]string)[0] = "changed"

	require.NoError(t, rt.TrackEmptyEventWithAdditionalProps(identity, open, opts...))
	open["list"].([]any)[0] = "changed"
	open["added"] = true
	ctx.IP[0] = 10
	ctx.Extra["custom"].(map[string]any)["k"] = "changed"
	ctx.Traits["plan"].([]any)[0] = "changed"
	integrations["Amplitude"].(map[string]any)["key"] = "changed"
	integrations["All"] = false
}

// The golden's own event methods are called, so the test covers the whole
// generated call path: payload conversion, options and the send helper.
func TestGeneratedMethodsOwnTheirInputs(t *testing.T) {
	client := &fakeClient{}
	sendAndChange(t, client)

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

// The real SDK serializes each message on its own goroutine right after
// Enqueue, so the changes race with it unless the call copied its inputs,
// and -race reports that.
func TestSDKSendsInputsAsPassed(t *testing.T) {
	wire, _ := capture(t, func(client analytics.Client) { sendAndChange(t, client) })

	require.Len(t, wire, 2)
	assertMessage(t, `{"type": "track", "channel": "server", "event": "User Signed Up", "userId": "user-123",
		"properties": {
			"array_of_any": ["x", ["y"]],
			"mixed_unicode": "a",
			"object_property": {"nested": {"k": "v"}},
			"property_of_any": ["z"]
		},
		"context": {"ip": "127.0.0.1", "custom": {"k": "v"}, "traits": {"plan": ["pro"]}, "ruddertyper": `+refMeta+`},
		"integrations": {"Amplitude": {"key": "v"}}}`, wire[0])
	assertMessage(t, `{"type": "track", "channel": "server", "event": "Empty Event With Additional Props", "userId": "user-123",
		"properties": {"list": ["x"]},
		"context": {"ip": "127.0.0.1", "custom": {"k": "v"}, "traits": {"plan": ["pro"]}, "ruddertyper": `+refMeta+`},
		"integrations": {"Amplitude": {"key": "v"}}}`, wire[1])
}

// The goroutines also share one option, as callers that build their options
// once do, so -race reports any call that writes to what it shares.
func TestConcurrentSends(t *testing.T) {
	const n = 100
	wire, _ := capture(t, func(client analytics.Client) {
		var (
			rt  = ruddertyper.New(client)
			opt = ruddertyper.WithAnalyticsContext(analytics.Context{Extra: map[string]any{"app": "validator"}})
			wg  sync.WaitGroup
		)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				assert.NoError(t, rt.TrackEmptyEventWithAdditionalProps(
					ruddertyper.Identity{UserID: fmt.Sprint("user-", i)}, map[string]any{"n": i}, opt,
				))
			}(i)
		}
		wg.Wait()
	})

	require.Len(t, wire, n)
	byUser := make(map[any]map[string]any, n)
	for _, msg := range wire {
		byUser[msg["userId"]] = msg
	}
	for i := 0; i < n; i++ {
		assertMessage(t, fmt.Sprintf(`{"type": "track", "channel": "server", "event": "Empty Event With Additional Props",
			"userId": "user-%d", "properties": {"n": %d},
			"context": {"app": "validator", "ruddertyper": %s}}`, i, i, refMeta), byUser[fmt.Sprint("user-", i)])
	}
}
