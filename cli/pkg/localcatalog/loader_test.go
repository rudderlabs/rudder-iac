package localcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractCatalogEntity(t *testing.T) {
	emptyCatalog := DataCatalog{
		Events:        make(map[EntityGroup][]*Event),
		Properties:    make(map[EntityGroup][]*Property),
		TrackingPlans: make(map[EntityGroup]*TrackingPlan),
	}

	t.Run("properties are extracted from customer defined yaml successfully", func(t *testing.T) {

		byt := []byte(`
        version: rudder/0.1
        kind: properties
        metadata:
          name: base_props
        spec:
          properties:
            - id: write_key
              display_name: "Write Key"
              type: string
              description: KSUID identifier for the source embedded in the SDKs
              propConfig:
                minLength: 24
                maxLength: 28
        `)

		err := extractEntities(byt, &emptyCatalog)
		require.Nil(t, err)
		assert.Equal(t, len(emptyCatalog.Properties), 1)
		assert.Equal(t, len(emptyCatalog.Properties["base_props"]), 1)
		assert.Equal(t, &Property{
			LocalID:     "write_key",
			Name:        "Write Key",
			Type:        "string",
			Description: "KSUID identifier for the source embedded in the SDKs",
			Config: map[string]interface{}{
				"minLength": float64(24),
				"maxLength": float64(28),
			},
		}, emptyCatalog.Properties["base_props"][0])
	})

	t.Run("events are extracted from customer defined yamls successfully", func(t *testing.T) {

		byt := []byte(`
        version: rudder/0.1
        kind: events
        metadata:
          name: app_events
        spec:
          events:
            - id: user_signed_up
              display_name: "User Signed Up"
              event_type: track
              description: "Triggered when user successfully signed up"
              categories:
                - "User Onboarding"
                - "Marketing Team"
        `)

		err := extractEntities(byt, &emptyCatalog)
		require.Nil(t, err)

		assert.Equal(t, 1, len(emptyCatalog.Events))
		assert.Equal(t, 1, len(emptyCatalog.Events["app_events"]))
		assert.Equal(t, &Event{
			LocalID:     "user_signed_up",
			Name:        "User Signed Up",
			Type:        "track",
			Description: "Triggered when user successfully signed up",
			Categories:  []string{"User Onboarding", "Marketing Team"},
		}, emptyCatalog.Events["app_events"][0])
	})

	t.Run("tracking plan entities are extracted from yaml successfully", func(t *testing.T) {

		byt := []byte(`
        version: rudder/0.1
        kind: events
        metadata:
          name: mobile_events
        spec:
          events:
            - id: user_signed_up
              display_name: "User Signed Up"
              event_type: track
              description: "Triggered when user successfully signed up"
              categories:
                - "User Onboarding"
                - "Marketing Team"
        `)

		err := extractEntities(byt, &emptyCatalog)
		require.Nil(t, err)

		byt = []byte(`
        version: rudder/0.1
        kind: properties
        metadata:
          name: base_mobile_props
        spec:
          properties:
            - id: username
              display_name: "Username of the customer"
              type: string
              description: "Username of the customer used for login"
              propConfig:
                minLength: 10
                maxLength: 63
            - id: button_signin
              display_name: "Button used for signin in the app"
              type: string
              description: "Button used for signin in the app"
              propConfig:
                enum: '["Sign In", "Sign Up"]'
       `)

		err = extractEntities(byt, &emptyCatalog)
		require.Nil(t, err)

		byt = []byte(`
        version: rudder/0.1
        kind: tp
        metadata:
          name: my_first_tp
        spec:
          id: my_first_tp
          display_name: "Rudderstack First Tracking Plan"
          description: "This is my first tracking plan"
          allow_unplanned: false
          rules:
            - type: event_rule
              id: rule_01
              event:
                $ref: "#/events/mobile_events/user_signed_up"
                allow_unplanned: true
              properties:
                - $ref: "#/properties/base_mobile_props/username"
                  required: true
                - $ref: "#/properties/base_mobile_props/button_signin"
                  required: false
        `)

		err = extractEntities(byt, &emptyCatalog)
		require.Nil(t, err)

		require.Equal(t, 1, len(emptyCatalog.TrackingPlans))
		assert.Equal(t, &TrackingPlan{
			Name:        "Rudderstack First Tracking Plan",
			LocalID:     "my_first_tp",
			Description: "This is my first tracking plan",
			Rules: []*TPRule{
				{
					LocalID: "rule_01",
					Type:    "event_rule",
					Event: &TPRuleEvent{
						Ref:            "#/events/mobile_events/user_signed_up",
						AllowUnplanned: true,
					},
					Properties: []*TPRuleProperty{
						{
							Ref:      "#/properties/base_mobile_props/username",
							Required: true,
						},
						{
							Ref:      "#/properties/base_mobile_props/button_signin",
							Required: false,
						},
					},
				},
			},
		}, emptyCatalog.TrackingPlans["my_first_tp"])
	})

}
