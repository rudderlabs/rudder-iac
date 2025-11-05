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

	t.Run("events defined in separate files with the same metadata.name should merge correctly", func(t *testing.T) {
		catalog := DataCatalog{
			Events:        make(map[EntityGroup][]Event),
			Properties:    make(map[EntityGroup][]Property),
			TrackingPlans: make(map[EntityGroup]*TrackingPlan),
			CustomTypes:   make(map[EntityGroup][]CustomType),
			Categories:    make(map[EntityGroup][]Category),
		}

		// First events file with metadata.name "shared_events"
		byt1 := []byte(`
        version: rudder/0.1
        kind: events
        metadata:
          name: shared_events
        spec:
          events:
            - id: event_a
              name: "Event A"
              event_type: track
              description: "First event from file 1"
            - id: event_b
              name: "Event B"
              event_type: track
              description: "Second event from file 1"
        `)

		s1, err := specs.New(byt1)
		require.Nil(t, err)
		err = extractEntities(s1, &catalog)
		require.Nil(t, err)

		// Verify first file loaded correctly
		assert.Len(t, catalog.Events["shared_events"], 2)
		assert.Equal(t, "event_a", catalog.Events["shared_events"][0].LocalID)
		assert.Equal(t, "event_b", catalog.Events["shared_events"][1].LocalID)

		// Second events file with same metadata.name "shared_events"
		byt2 := []byte(`
        version: rudder/0.1
        kind: events
        metadata:
          name: shared_events
        spec:
          events:
            - id: event_c
              name: "Event C"
              event_type: track
              description: "First event from file 2"
            - id: event_d
              name: "Event D"
              event_type: track
              description: "Second event from file 2"
        `)

		s2, err := specs.New(byt2)
		require.Nil(t, err)
		err = extractEntities(s2, &catalog)
		require.Nil(t, err)

		// Verify both files are merged - should have 4 events total
		assert.Len(t, catalog.Events["shared_events"], 4)
		assert.Equal(t, "event_a", catalog.Events["shared_events"][0].LocalID)
		assert.Equal(t, "event_b", catalog.Events["shared_events"][1].LocalID)
		assert.Equal(t, "event_c", catalog.Events["shared_events"][2].LocalID)
		assert.Equal(t, "event_d", catalog.Events["shared_events"][3].LocalID)
	})

	t.Run("properties defined in separate files with the same metadata.name should merge correctly", func(t *testing.T) {
		catalog := DataCatalog{
			Events:        make(map[EntityGroup][]Event),
			Properties:    make(map[EntityGroup][]Property),
			TrackingPlans: make(map[EntityGroup]*TrackingPlan),
			CustomTypes:   make(map[EntityGroup][]CustomType),
			Categories:    make(map[EntityGroup][]Category),
		}

		// First properties file
		byt1 := []byte(`
        version: rudder/0.1
        kind: properties
        metadata:
          name: shared_props
        spec:
          properties:
            - id: prop_a
              name: "Property A"
              type: string
              description: "First property from file 1"
            - id: prop_b
              name: "Property B"
              type: integer
              description: "Second property from file 1"
        `)

		s1, err := specs.New(byt1)
		require.Nil(t, err)
		err = extractEntities(s1, &catalog)
		require.Nil(t, err)

		// Second properties file with same metadata.name
		byt2 := []byte(`
        version: rudder/0.1
        kind: properties
        metadata:
          name: shared_props
        spec:
          properties:
            - id: prop_c
              name: "Property C"
              type: boolean
              description: "First property from file 2"
        `)

		s2, err := specs.New(byt2)
		require.Nil(t, err)
		err = extractEntities(s2, &catalog)
		require.Nil(t, err)

		// Verify both files are merged - should have 3 properties total
		assert.Len(t, catalog.Properties["shared_props"], 3)
		assert.Equal(t, "prop_a", catalog.Properties["shared_props"][0].LocalID)
		assert.Equal(t, "prop_b", catalog.Properties["shared_props"][1].LocalID)
		assert.Equal(t, "prop_c", catalog.Properties["shared_props"][2].LocalID)
	})

	t.Run("categories defined in separate files with the same metadata.name should merge correctly", func(t *testing.T) {
		catalog := DataCatalog{
			Events:        make(map[EntityGroup][]Event),
			Properties:    make(map[EntityGroup][]Property),
			TrackingPlans: make(map[EntityGroup]*TrackingPlan),
			CustomTypes:   make(map[EntityGroup][]CustomType),
			Categories:    make(map[EntityGroup][]Category),
		}

		// First categories file
		byt1 := []byte(`
        version: rudder/0.1
        kind: categories
        metadata:
          name: shared_categories
        spec:
          categories:
            - id: cat_a
              name: "Category A"
            - id: cat_b
              name: "Category B"
        `)

		s1, err := specs.New(byt1)
		require.Nil(t, err)
		err = extractEntities(s1, &catalog)
		require.Nil(t, err)

		// Second categories file with same metadata.name
		byt2 := []byte(`
        version: rudder/0.1
        kind: categories
        metadata:
          name: shared_categories
        spec:
          categories:
            - id: cat_c
              name: "Category C"
        `)

		s2, err := specs.New(byt2)
		require.Nil(t, err)
		err = extractEntities(s2, &catalog)
		require.Nil(t, err)

		// Verify both files are merged - should have 3 categories total
		assert.Len(t, catalog.Categories["shared_categories"], 3)
		assert.Equal(t, "cat_a", catalog.Categories["shared_categories"][0].LocalID)
		assert.Equal(t, "cat_b", catalog.Categories["shared_categories"][1].LocalID)
		assert.Equal(t, "cat_c", catalog.Categories["shared_categories"][2].LocalID)
	})

	t.Run("custom-types defined in separate files with the same metadata.name should merge correctly", func(t *testing.T) {
		catalog := DataCatalog{
			Events:        make(map[EntityGroup][]Event),
			Properties:    make(map[EntityGroup][]Property),
			TrackingPlans: make(map[EntityGroup]*TrackingPlan),
			CustomTypes:   make(map[EntityGroup][]CustomType),
			Categories:    make(map[EntityGroup][]Category),
		}

		// First custom-types file
		byt1 := []byte(`
        version: rudder/0.1
        kind: custom-types
        metadata:
          name: shared_types
        spec:
          types:
            - id: type_a
              name: "Type A"
              type: string
              description: "First type from file 1"
            - id: type_b
              name: "Type B"
              type: integer
              description: "Second type from file 1"
        `)

		s1, err := specs.New(byt1)
		require.Nil(t, err)
		err = extractEntities(s1, &catalog)
		require.Nil(t, err)

		// Second custom-types file with same metadata.name
		byt2 := []byte(`
        version: rudder/0.1
        kind: custom-types
        metadata:
          name: shared_types
        spec:
          types:
            - id: type_c
              name: "Type C"
              type: boolean
              description: "First type from file 2"
        `)

		s2, err := specs.New(byt2)
		require.Nil(t, err)
		err = extractEntities(s2, &catalog)
		require.Nil(t, err)

		// Verify both files are merged - should have 3 custom types total
		assert.Len(t, catalog.CustomTypes["shared_types"], 3)
		assert.Equal(t, "type_a", catalog.CustomTypes["shared_types"][0].LocalID)
		assert.Equal(t, "type_b", catalog.CustomTypes["shared_types"][1].LocalID)
		assert.Equal(t, "type_c", catalog.CustomTypes["shared_types"][2].LocalID)
	})

	t.Run("duplicate tracking plan metadata.name should return error", func(t *testing.T) {
		catalog := DataCatalog{
			Events:        make(map[EntityGroup][]Event),
			Properties:    make(map[EntityGroup][]Property),
			TrackingPlans: make(map[EntityGroup]*TrackingPlan),
			CustomTypes:   make(map[EntityGroup][]CustomType),
			Categories:    make(map[EntityGroup][]Category),
		}

		// First tracking plan file
		byt1 := []byte(`
        version: rudder/0.1
        kind: tp
        metadata:
          name: shared_tp
        spec:
          id: tp_1
          display_name: "Tracking Plan 1"
          description: "First tracking plan"
          rules: []
        `)

		s1, err := specs.New(byt1)
		require.Nil(t, err)
		err = extractEntities(s1, &catalog)
		require.Nil(t, err)

		// Verify first tracking plan loaded correctly
		assert.Len(t, catalog.TrackingPlans, 1)
		assert.NotNil(t, catalog.TrackingPlans["shared_tp"])

		// Second tracking plan file with same metadata.name should fail
		byt2 := []byte(`
        version: rudder/0.1
        kind: tp
        metadata:
          name: shared_tp
        spec:
          id: tp_2
          display_name: "Tracking Plan 2"
          description: "Second tracking plan"
          rules: []
        `)

		s2, err := specs.New(byt2)
		require.Nil(t, err)
		err = extractEntities(s2, &catalog)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate tracking plan with metadata.name 'shared_tp' found")
		assert.Contains(t, err.Error(), "only one tracking plan per entity group is allowed")
	})
}

func TestDataCatalog_ParseSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		spec           *specs.Spec
		expectedIDs    []string
		expectedError  bool
		errorContains  string
	}{
		{
			name: "success - parse properties spec with multiple IDs",
			spec: &specs.Spec{
				Kind: KindProperties,
				Spec: map[string]any{
					"properties": []any{
						map[string]any{"id": "prop1", "name": "Property 1"},
						map[string]any{"id": "prop2", "name": "Property 2"},
						map[string]any{"id": "prop3", "name": "Property 3"},
					},
				},
			},
			expectedIDs:   []string{"prop1", "prop2", "prop3"},
			expectedError: false,
		},
		{
			name: "success - parse events spec with multiple IDs",
			spec: &specs.Spec{
				Kind: KindEvents,
				Spec: map[string]any{
					"events": []any{
						map[string]any{"id": "event1", "name": "Event 1"},
						map[string]any{"id": "event2", "name": "Event 2"},
					},
				},
			},
			expectedIDs:   []string{"event1", "event2"},
			expectedError: false,
		},
		{
			name: "success - parse tracking plan spec",
			spec: &specs.Spec{
				Kind: KindTrackingPlans,
				Spec: map[string]any{
					"id":           "my_tracking_plan",
					"display_name": "My Tracking Plan",
				},
			},
			expectedIDs:   []string{"my_tracking_plan"},
			expectedError: false,
		},
		{
			name: "success - parse custom types spec with multiple IDs",
			spec: &specs.Spec{
				Kind: KindCustomTypes,
				Spec: map[string]any{
					"types": []any{
						map[string]any{"id": "type1", "name": "Type 1"},
						map[string]any{"id": "type2", "name": "Type 2"},
					},
				},
			},
			expectedIDs:   []string{"type1", "type2"},
			expectedError: false,
		},
		{
			name: "success - parse categories spec with multiple IDs",
			spec: &specs.Spec{
				Kind: KindCategories,
				Spec: map[string]any{
					"categories": []any{
						map[string]any{"id": "cat1", "name": "Category 1"},
						map[string]any{"id": "cat2", "name": "Category 2"},
						map[string]any{"id": "cat3", "name": "Category 3"},
					},
				},
			},
			expectedIDs:   []string{"cat1", "cat2", "cat3"},
			expectedError: false,
		},
		{
			name: "error - properties not found in spec",
			spec: &specs.Spec{
				Kind: KindProperties,
				Spec: map[string]any{
					"other": "value",
				},
			},
			expectedError: true,
			errorContains: "properties not found in spec",
		},
		{
			name: "error - properties is not an array",
			spec: &specs.Spec{
				Kind: KindProperties,
				Spec: map[string]any{
					"properties": "not_an_array",
				},
			},
			expectedError: true,
			errorContains: "properties not found in spec",
		},
		{
			name: "error - events not found in spec",
			spec: &specs.Spec{
				Kind: KindEvents,
				Spec: map[string]any{
					"other": "value",
				},
			},
			expectedError: true,
			errorContains: "events not found in spec",
		},
		{
			name: "error - events is not an array",
			spec: &specs.Spec{
				Kind: KindEvents,
				Spec: map[string]any{
					"events": "not_an_array",
				},
			},
			expectedError: true,
			errorContains: "events not found in spec",
		},
		{
			name: "error - tracking plan id not found",
			spec: &specs.Spec{
				Kind: KindTrackingPlans,
				Spec: map[string]any{
					"display_name": "My TP",
				},
			},
			expectedError: true,
			errorContains: "id not found in tracking plan spec",
		},
		{
			name: "error - tracking plan id is not a string",
			spec: &specs.Spec{
				Kind: KindTrackingPlans,
				Spec: map[string]any{
					"id": 12345,
				},
			},
			expectedError: true,
			errorContains: "id not found in tracking plan spec",
		},
		{
			name: "error - custom types not found in spec",
			spec: &specs.Spec{
				Kind: KindCustomTypes,
				Spec: map[string]any{
					"other": "value",
				},
			},
			expectedError: true,
			errorContains: "custom types not found in spec",
		},
		{
			name: "error - custom types is not an array",
			spec: &specs.Spec{
				Kind: KindCustomTypes,
				Spec: map[string]any{
					"types": "not_an_array",
				},
			},
			expectedError: true,
			errorContains: "custom types not found in spec",
		},
		{
			name: "error - categories not found in spec",
			spec: &specs.Spec{
				Kind: KindCategories,
				Spec: map[string]any{
					"other": "value",
				},
			},
			expectedError: true,
			errorContains: "categories not found in spec",
		},
		{
			name: "error - categories is not an array",
			spec: &specs.Spec{
				Kind: KindCategories,
				Spec: map[string]any{
					"categories": "not_an_array",
				},
			},
			expectedError: true,
			errorContains: "categories not found in spec",
		},
		{
			name: "error - entity is not a map",
			spec: &specs.Spec{
				Kind: KindProperties,
				Spec: map[string]any{
					"properties": []any{
						"not_a_map",
					},
				},
			},
			expectedError: true,
			errorContains: "entity is not a map[string]any",
		},
		{
			name: "error - id field not found in entity",
			spec: &specs.Spec{
				Kind: KindProperties,
				Spec: map[string]any{
					"properties": []any{
						map[string]any{"name": "Property without ID"},
					},
				},
			},
			expectedError: true,
			errorContains: "id not found in entity",
		},
		{
			name: "error - id field is not a string",
			spec: &specs.Spec{
				Kind: KindEvents,
				Spec: map[string]any{
					"events": []any{
						map[string]any{"id": 12345},
					},
				},
			},
			expectedError: true,
			errorContains: "id not found in entity",
		},
		{
			name: "success - empty properties array",
			spec: &specs.Spec{
				Kind: KindProperties,
				Spec: map[string]any{
					"properties": []any{},
				},
			},
			expectedIDs:   nil,
			expectedError: false,
		},
		{
			name: "success - empty events array",
			spec: &specs.Spec{
				Kind: KindEvents,
				Spec: map[string]any{
					"events": []any{},
				},
			},
			expectedIDs:   nil,
			expectedError: false,
		},
		{
			name: "success - empty custom types array",
			spec: &specs.Spec{
				Kind: KindCustomTypes,
				Spec: map[string]any{
					"types": []any{},
				},
			},
			expectedIDs:   nil,
			expectedError: false,
		},
		{
			name: "success - empty categories array",
			spec: &specs.Spec{
				Kind: KindCategories,
				Spec: map[string]any{
					"categories": []any{},
				},
			},
			expectedIDs:   nil,
			expectedError: false,
		},
	}

	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dc := New()
			parsedSpec, err := dc.ParseSpec("test/path.yaml", tc.spec)

			if tc.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, parsedSpec)
			} else {
				require.NoError(t, err)
				require.NotNil(t, parsedSpec)
				assert.Equal(t, tc.expectedIDs, parsedSpec.ExternalIDs)
			}
		})
	}
}
