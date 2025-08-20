package localcatalog

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractCatalogEntity(t *testing.T) {
	emptyCatalog := DataCatalog{
		Events:        make(map[EntityGroup][]Event),
		Properties:    make(map[EntityGroup][]Property),
		TrackingPlans: make(map[EntityGroup]*TrackingPlan),
		CustomTypes:   make(map[EntityGroup][]CustomType),
		Categories:    make(map[EntityGroup][]Category),
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
              name: "Write Key"
              type: string
              description: KSUID identifier for the source embedded in the SDKs
              propConfig:
                minLength: 24
                maxLength: 28
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)
		assert.Equal(t, len(emptyCatalog.Properties), 1)
		assert.Equal(t, len(emptyCatalog.Properties["base_props"]), 1)
		assert.Equal(t, Property{
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

		category := "#/categories/app_categories/user_actions"
		byt := []byte(`
        version: rudder/0.1
        kind: events
        metadata:
          name: app_events
        spec:
          events:
            - id: user_signed_up
              name: "User Signed Up"
              event_type: track
              description: "Triggered when user successfully signed up"
              category: "#/categories/app_categories/user_actions"
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		assert.Equal(t, 1, len(emptyCatalog.Events))
		assert.Equal(t, 1, len(emptyCatalog.Events["app_events"]))
		assert.Equal(t, Event{
			LocalID:     "user_signed_up",
			Name:        "User Signed Up",
			Type:        "track",
			Description: "Triggered when user successfully signed up",
			CategoryRef: &category,
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
              name: "User Signed Up"
              event_type: track
              description: "Triggered when user successfully signed up"
              category: "#/categories/app_categories/user_actions"
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		byt = []byte(`
        version: rudder/0.1
        kind: properties
        metadata:
          name: base_mobile_props
        spec:
          properties:
            - id: username
              name: "Username of the customer"
              type: string
              description: "Username of the customer used for login"
              propConfig:
                minLength: 10
                maxLength: 63
            - id: button_signin
              name: "Button used for signin in the app"
              type: object
              description: "Button used for signin in the app"
            - id: remember_me_checkbox_clicked
              name: "Remember Me Checkbox Clicked"
              type: boolean
              description: "Whether the remember me checkbox was clicked during signin"
            - id: captcha
              name: "Captcha"
              type: object
              description: "Captcha details during signin"
            - id: captcha_solved
              name: "Captcha Solved"
              type: boolean
              description: "Whether the captcha was solved during signin"
            - id: captcha_type
              name: "Captcha Type"
              type: string
              description: "Type of captcha used during signin"
       `)

		s, err = specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
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
                  properties:
                    - $ref: "#/properties/base_mobile_props/username"
                      required: true
                    - $ref: "#/properties/base_mobile_props/remember_me_checkbox_clicked"
                      required: false
                    - $ref: "#/properties/base_mobile_props/captcha"
                      required: false
                      properties:
                        - $ref: "#/properties/base_mobile_props/captcha_solved"
                          required: false
                        - $ref: "#/properties/base_mobile_props/captcha_type"
                          required: false
        `)

		s, err = specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		require.Equal(t, 1, len(emptyCatalog.TrackingPlans))
		assert.Equal(t, TrackingPlan{
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
							Properties: []*TPRuleProperty{
								{
									Ref:      "#/properties/base_mobile_props/username",
									Required: true,
								},
								{
									Ref:      "#/properties/base_mobile_props/remember_me_checkbox_clicked",
									Required: false,
								},
								{
									Ref:      "#/properties/base_mobile_props/captcha",
									Required: false,
									Properties: []*TPRuleProperty{
										{
											Ref:      "#/properties/base_mobile_props/captcha_solved",
											Required: false,
										},
										{
											Ref:      "#/properties/base_mobile_props/captcha_type",
											Required: false,
										},
									},
								},
							},
						},
					},
				},
			},
		}, *emptyCatalog.TrackingPlans["my_first_tp"])

		byt = []byte(`
        version: rudder/0.1
        kind: tp
        metadata:
          description: "This is my first tracking plan"
        spec:
          events:
            - id: user_signed_up
              name: "User Signed Up"
              event_type: track
              description: "Triggered when user successfully signed up"
              category: "#/categories/app_categories/user_actions"
        `)

		s, err = specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)
	})

	t.Run("tracking plan with variants is extracted successfully", func(t *testing.T) {

		byt := []byte(`
        version: rudder/v0.1
        kind: tp
        metadata:
          name: tp_with_variants
        spec:
          id: tp_with_variants
          display_name: "tracking plan with variants"
          description: "testing variants field support"
          rules:
            - type: event_rule
              id: rule_with_variants
              event:
                $ref: "#/events/mobile_events/user_signed_up"
              properties:
                - $ref: "#/properties/mypropertygroup/page_name"
                  required: true
              variants:
                - type: discriminator
                  discriminator: "page_name"
                  cases:
                    - display_name: "Search Page"
                      match:
                      - "search"
                      - "search_bar"
                      description: "applies when a product is viewed as part of search results"
                      properties:
                      - $ref: "#/properties/mypropertygroup/search_term"
                        required: true
                    - "display_name": "Product Page"
                      match: 
                      - "product"
                      - "search"
                      - "1"
                      properties:
                      - $ref: "#/properties/mypropertygroup/product_id"
                        required: true
                  default:
                     - $ref: "#/properties/mypropertygroup/page_url"
                       required: true
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		tp := emptyCatalog.TrackingPlans["tp_with_variants"]
		require.NotNil(t, tp)
		require.Equal(t, 1, len(tp.Rules))

		rule := tp.Rules[0]
		require.NotNil(t, rule.Variants)
		require.Equal(t, 1, len(rule.Variants))

		variant := (rule.Variants)[0]
		assert.Equal(t, "discriminator", variant.Type)
		assert.Equal(t, "page_name", variant.Discriminator)
		assert.Equal(t, 2, len(variant.Cases))
		assert.Equal(t, "Search Page", variant.Cases[0].DisplayName)
		assert.Equal(t, []any{"search", "search_bar"}, variant.Cases[0].Match)
		assert.Equal(t, "Product Page", variant.Cases[1].DisplayName)
		assert.Equal(t, []any{"product", "search", "1"}, variant.Cases[1].Match)
		assert.Equal(t, 1, len(variant.Default))
	})

	t.Run("custom types are extracted from customer defined yaml successfully", func(t *testing.T) {

		byt := []byte(`
        version: "rudder/v0.1"
        kind: "custom-types"
        metadata:
          name: "email-types"
        spec:
          types:
            - id: "EmailType"
              name: "Email Type"
              description: "Custom type for email validation"
              type: "string"
              config:
                format: "email"
                minLength: 5
                maxLength: 255
                pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
            - id: "ProductIdType"
              name: "Product ID Type"
              description: "Custom type for product identifiers"
              type: "string"
              config:
                minLength: 10
                maxLength: 20
                pattern: "^PROD-[0-9]{7}$"
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)
		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		assert.Equal(t, 1, len(emptyCatalog.CustomTypes))
		assert.Equal(t, 2, len(emptyCatalog.CustomTypes["email-types"]))

		// Verify first custom type (EmailType)
		assert.Equal(t, CustomType{
			LocalID:     "EmailType",
			Name:        "Email Type",
			Description: "Custom type for email validation",
			Type:        "string",
			Config: map[string]interface{}{
				"format":    "email",
				"minLength": float64(5),
				"maxLength": float64(255),
				"pattern":   "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
			},
		}, emptyCatalog.CustomTypes["email-types"][0])

		// Verify second custom type (ProductIdType)
		assert.Equal(t, CustomType{
			LocalID:     "ProductIdType",
			Name:        "Product ID Type",
			Description: "Custom type for product identifiers",
			Type:        "string",
			Config: map[string]interface{}{
				"minLength": float64(10),
				"maxLength": float64(20),
				"pattern":   "^PROD-[0-9]{7}$",
			},
		}, emptyCatalog.CustomTypes["email-types"][1])
	})

	t.Run("custom type with variants field is extracted successfully", func(t *testing.T) {

		byt := []byte(`
        version: "rudder/v0.1"
        kind: "custom-types"
        metadata:
          name: "custom-types-with-variants"
        spec:
          types:
            - id: "user_profile_type"
              name: "user profile type"
              description: "custom object type with variants support"
              type: "object"
              properties:
                - $ref: "#/properties/profiles/profile_type"
                  required: true
              variants:
                - type: discriminator
                  discriminator: "profile_type"
                  cases:
                    - display_name: "Premium User"
                      match:
                        - "premium"
                        - "vip"
                      properties:
                        - $ref: "#/properties/profiles/subscription_tier"
                          required: true
                    - display_name: "Basic User"
                      match:
                        - "basic"
                        - "free"
                      properties:
                        - $ref: "#/properties/profiles/usage_limit"
                          required: true
                  default:
                    - $ref: "#/properties/profiles/user_type"
                      required: true
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)
		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		assert.Equal(t, 2, len(emptyCatalog.CustomTypes))
		assert.Equal(t, 1, len(emptyCatalog.CustomTypes["custom-types-with-variants"]))

		customType := emptyCatalog.CustomTypes["custom-types-with-variants"][0]
		assert.Equal(t, "user_profile_type", customType.LocalID)
		assert.Equal(t, "object", customType.Type)
		assert.Equal(t, 1, len(customType.Variants))

		variant := customType.Variants[0]
		assert.Equal(t, "discriminator", variant.Type)
		assert.Equal(t, "profile_type", variant.Discriminator)
		assert.Equal(t, 2, len(variant.Cases))
		assert.Equal(t, "Premium User", variant.Cases[0].DisplayName)
		assert.Equal(t, []any{"premium", "vip"}, variant.Cases[0].Match)
		assert.Equal(t, "Basic User", variant.Cases[1].DisplayName)
		assert.Equal(t, []any{"basic", "free"}, variant.Cases[1].Match)
		assert.Equal(t, 1, len(variant.Default))
	})

	t.Run("custom types with property references are extracted successfully", func(t *testing.T) {

		byt := []byte(`
        version: "rudder/v0.1"
        kind: "custom-types"
        metadata:
          name: "object-types"
        spec:
          types:
            - id: "UserAddressType"
              name: "User Address Type"
              description: "Custom type for user address information"
              type: "object"
              properties: [
                { $ref: "#/properties/address/street", required: true },
                { $ref: "#/properties/address/city", required: true },
                { $ref: "#/properties/address/state", required: false },
                { $ref: "#/properties/address/zip", required: true }
              ]
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)
		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		assert.Equal(t, 3, len(emptyCatalog.CustomTypes)) // email-types + object-types-with-variants + object-types
		assert.Equal(t, 1, len(emptyCatalog.CustomTypes["object-types"]))

		// Verify object custom type with properties
		customType := emptyCatalog.CustomTypes["object-types"][0]
		assert.Equal(t, "UserAddressType", customType.LocalID)
		assert.Equal(t, "User Address Type", customType.Name)
		assert.Equal(t, "Custom type for user address information", customType.Description)
		assert.Equal(t, "object", customType.Type)

		// Verify properties array
		require.Equal(t, 4, len(customType.Properties))

		// Check each property reference
		assert.Equal(t, CustomTypeProperty{Ref: "#/properties/address/street", Required: true}, customType.Properties[0])
		assert.Equal(t, CustomTypeProperty{Ref: "#/properties/address/city", Required: true}, customType.Properties[1])
		assert.Equal(t, CustomTypeProperty{Ref: "#/properties/address/state", Required: false}, customType.Properties[2])
		assert.Equal(t, CustomTypeProperty{Ref: "#/properties/address/zip", Required: true}, customType.Properties[3])
	})

	t.Run("categories are extracted from customer defined yaml successfully", func(t *testing.T) {

		byt := []byte(`
        version: rudder/0.1
        kind: categories
        metadata:
          name: app_categories
        spec:
          categories:
            - id: user_actions
              name: "User Actions"
            - id: system_events
              name: "System Events"
            - id: payment_events
              name: "Payment Events"
        `)

		s, err := specs.New(byt)
		require.Nil(t, err)

		err = extractEntities(s, &emptyCatalog)
		require.Nil(t, err)

		assert.Equal(t, 1, len(emptyCatalog.Categories))
		assert.Equal(t, 3, len(emptyCatalog.Categories["app_categories"]))

		// Verify first category
		assert.Equal(t, Category{
			LocalID: "user_actions",
			Name:    "User Actions",
		}, emptyCatalog.Categories["app_categories"][0])

		// Verify second category
		assert.Equal(t, Category{
			LocalID: "system_events",
			Name:    "System Events",
		}, emptyCatalog.Categories["app_categories"][1])

		// Verify third category
		assert.Equal(t, Category{
			LocalID: "payment_events",
			Name:    "Payment Events",
		}, emptyCatalog.Categories["app_categories"][2])
	})
}
