package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYAMLMarshalingComprehensive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category  string
		name      string
		setup     func() interface{}
		validate  func(t *testing.T, original interface{}, yamlData []byte)
		unmarshal func(t *testing.T, yamlData string) interface{}
	}{
		// Events YAML Tests
		{
			category: "Events",
			name:     "MarshalToYAML",
			setup: func() interface{} {
				return EventsYAML{
					Version:  "rudder/v0.1",
					Kind:     "events",
					Metadata: YAMLMetadata{Name: "test_events"},
					Spec: EventsSpec{
						Events: []EventDefinition{
							{ID: "product_viewed", Name: "Product Viewed", EventType: "track", Description: "User viewed a product"},
							{ID: "user_signup", EventType: "identify"},
						},
					},
				}
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				yamlString := string(yamlData)
				assert.Contains(t, yamlString, "version: rudder/v0.1")
				assert.Contains(t, yamlString, "kind: events")
				assert.Contains(t, yamlString, "name: test_events")
				assert.Contains(t, yamlString, "product_viewed")
				assert.Contains(t, yamlString, "Product Viewed")
				assert.Contains(t, yamlString, "User viewed a product")
			},
		},
		{
			category: "Events",
			name:     "UnmarshalFromYAML",
			unmarshal: func(t *testing.T, yamlData string) interface{} {
				data := `
version: rudder/v0.1
kind: events
metadata:
  name: extracted_events
spec:
  events:
    - id: order_completed
      name: Order Completed
      event_type: track
      description: User completed an order
    - id: page_viewed
      event_type: page`

				var eventsYAML EventsYAML
				err := yaml.Unmarshal([]byte(data), &eventsYAML)
				require.NoError(t, err)
				return eventsYAML
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				eventsYAML := original.(EventsYAML)
				assert.Equal(t, "rudder/v0.1", eventsYAML.Version)
				assert.Equal(t, "events", eventsYAML.Kind)
				assert.Equal(t, "extracted_events", eventsYAML.Metadata.Name)
				assert.Len(t, eventsYAML.Spec.Events, 2)

				event1 := eventsYAML.Spec.Events[0]
				assert.Equal(t, "order_completed", event1.ID)
				assert.Equal(t, "Order Completed", event1.Name)
				assert.Equal(t, "track", event1.EventType)
				assert.Equal(t, "User completed an order", event1.Description)

				event2 := eventsYAML.Spec.Events[1]
				assert.Equal(t, "page_viewed", event2.ID)
				assert.Equal(t, "", event2.Name)
				assert.Equal(t, "page", event2.EventType)
				assert.Equal(t, "", event2.Description)
			},
		},
		{
			category: "Events",
			name:     "RoundTripMarshaling",
			setup: func() interface{} {
				return EventsYAML{
					Version:  "rudder/v0.1",
					Kind:     "events",
					Metadata: YAMLMetadata{Name: "round_trip_test"},
					Spec: EventsSpec{
						Events: []EventDefinition{
							{ID: "test_event_1", Name: "Test Event 1", EventType: "track", Description: "First test event"},
						},
					},
				}
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				originalObj := original.(EventsYAML)
				var restored EventsYAML
				err := yaml.Unmarshal(yamlData, &restored)
				require.NoError(t, err)

				assert.Equal(t, originalObj.Version, restored.Version)
				assert.Equal(t, originalObj.Kind, restored.Kind)
				assert.Equal(t, originalObj.Metadata.Name, restored.Metadata.Name)
				assert.Len(t, restored.Spec.Events, 1)
				assert.Equal(t, originalObj.Spec.Events[0], restored.Spec.Events[0])
			},
		},

		// Properties YAML Tests
		{
			category: "Properties",
			name:     "MarshalToYAML",
			setup: func() interface{} {
				minLen, maxLen := 3, 50
				min, max := 0.0, 100.0
				return PropertiesYAML{
					Version:  "rudder/v0.1",
					Kind:     "properties",
					Metadata: YAMLMetadata{Name: "test_properties"},
					Spec: PropertiesSpec{
						Properties: []PropertyDefinition{
							{
								ID: "user_email", Name: "User Email", Type: "string", Description: "User's email address",
								PropConfig: &PropertyConfig{MinLength: &minLen, MaxLength: &maxLen, Pattern: "^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$"},
							},
							{
								ID: "user_age", Name: "User Age", Type: "number",
								PropConfig: &PropertyConfig{Minimum: &min, Maximum: &max},
							},
						},
					},
				}
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				yamlString := string(yamlData)
				assert.Contains(t, yamlString, "version: rudder/v0.1")
				assert.Contains(t, yamlString, "kind: properties")
				assert.Contains(t, yamlString, "user_email")
				assert.Contains(t, yamlString, "User Email")
				assert.Contains(t, yamlString, "propConfig")
				assert.Contains(t, yamlString, "minLength: 3")
				assert.Contains(t, yamlString, "maxLength: 50")
			},
		},
		{
			category: "Properties",
			name:     "UnmarshalFromYAML",
			unmarshal: func(t *testing.T, yamlData string) interface{} {
				data := `
version: rudder/v0.1
kind: properties
metadata:
  name: extracted_properties
spec:
  properties:
    - id: product_name
      name: Product Name
      type: string
      description: Name of the product
      propConfig:
        minLength: 1
        maxLength: 100
    - id: categories
      name: Categories
      type: array
      propConfig:
        minItems: 1
        maxItems: 10
        uniqueItems: true`

				var propertiesYAML PropertiesYAML
				err := yaml.Unmarshal([]byte(data), &propertiesYAML)
				require.NoError(t, err)
				return propertiesYAML
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				propertiesYAML := original.(PropertiesYAML)
				assert.Equal(t, "rudder/v0.1", propertiesYAML.Version)
				assert.Equal(t, "properties", propertiesYAML.Kind)
				assert.Equal(t, "extracted_properties", propertiesYAML.Metadata.Name)
				assert.Len(t, propertiesYAML.Spec.Properties, 2)

				prop1 := propertiesYAML.Spec.Properties[0]
				assert.Equal(t, "product_name", prop1.ID)
				assert.Equal(t, "Product Name", prop1.Name)
				assert.Equal(t, "string", prop1.Type)
				assert.Equal(t, "Name of the product", prop1.Description)
				require.NotNil(t, prop1.PropConfig)
				assert.Equal(t, 1, *prop1.PropConfig.MinLength)
				assert.Equal(t, 100, *prop1.PropConfig.MaxLength)

				prop2 := propertiesYAML.Spec.Properties[1]
				assert.Equal(t, "categories", prop2.ID)
				assert.Equal(t, "array", prop2.Type)
				require.NotNil(t, prop2.PropConfig)
				assert.Equal(t, 1, *prop2.PropConfig.MinItems)
				assert.Equal(t, 10, *prop2.PropConfig.MaxItems)
				assert.True(t, *prop2.PropConfig.UniqueItems)
			},
		},

		// Custom Types YAML Tests
		{
			category: "CustomTypes",
			name:     "MarshalToYAML",
			setup: func() interface{} {
				return CustomTypesYAML{
					Version:  "rudder/v0.1",
					Kind:     "custom-types",
					Metadata: YAMLMetadata{Name: "test_custom_types"},
					Spec: CustomTypesSpec{
						Types: []CustomTypeDefinition{
							{
								ID: "user_profile", Name: "UserProfile", Type: "object", Description: "User profile information",
								Properties: []PropertyRef{
									{Ref: "#/properties/user_email", Required: true},
									{Ref: "#/properties/user_name", Required: true},
									{Ref: "#/properties/user_age", Required: false},
								},
							},
						},
					},
				}
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				yamlString := string(yamlData)
				assert.Contains(t, yamlString, "version: rudder/v0.1")
				assert.Contains(t, yamlString, "kind: custom-types")
				assert.Contains(t, yamlString, "user_profile")
				assert.Contains(t, yamlString, "UserProfile")
				assert.Contains(t, yamlString, "User profile information")
				assert.Contains(t, yamlString, "#/properties/user_email")
				assert.Contains(t, yamlString, "required: true")
				assert.Contains(t, yamlString, "required: false")
			},
		},
		{
			category: "CustomTypes",
			name:     "UnmarshalFromYAML",
			unmarshal: func(t *testing.T, yamlData string) interface{} {
				data := `
version: rudder/v0.1
kind: custom-types
metadata:
  name: extracted_custom_types
spec:
  types:
    - id: address_type
      name: AddressType
      type: object
      description: Address information
      properties:
        - $ref: "#/properties/street"
          required: true
        - $ref: "#/properties/city"
          required: true
        - $ref: "#/properties/zipcode"
          required: false
    - id: product_array
      name: ProductArray
      type: array
      config:
        minItems: 1
        maxItems: 100
        itemTypes: ["#/custom-types/product_type"]`

				var customTypesYAML CustomTypesYAML
				err := yaml.Unmarshal([]byte(data), &customTypesYAML)
				require.NoError(t, err)
				return customTypesYAML
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				customTypesYAML := original.(CustomTypesYAML)
				assert.Equal(t, "rudder/v0.1", customTypesYAML.Version)
				assert.Equal(t, "custom-types", customTypesYAML.Kind)
				assert.Equal(t, "extracted_custom_types", customTypesYAML.Metadata.Name)
				assert.Len(t, customTypesYAML.Spec.Types, 2)

				type1 := customTypesYAML.Spec.Types[0]
				assert.Equal(t, "address_type", type1.ID)
				assert.Equal(t, "AddressType", type1.Name)
				assert.Equal(t, "object", type1.Type)
				assert.Equal(t, "Address information", type1.Description)
				assert.Len(t, type1.Properties, 3)
				assert.True(t, type1.Properties[0].Required)
				assert.True(t, type1.Properties[1].Required)
				assert.False(t, type1.Properties[2].Required)

				type2 := customTypesYAML.Spec.Types[1]
				assert.Equal(t, "product_array", type2.ID)
				assert.Equal(t, "ProductArray", type2.Name)
				assert.Equal(t, "array", type2.Type)
				require.NotNil(t, type2.Config)
				assert.Equal(t, 1, *type2.Config.MinItems)
				assert.Equal(t, 100, *type2.Config.MaxItems)
				assert.Len(t, type2.Config.ItemTypes, 1)
				assert.Equal(t, "#/custom-types/product_type", type2.Config.ItemTypes[0])
			},
		},

		// Tracking Plan YAML Tests
		{
			category: "TrackingPlan",
			name:     "MarshalToYAML",
			setup: func() interface{} {
				return TrackingPlanYAML{
					Version:  "rudder/v0.1",
					Kind:     "tracking-plan",
					Metadata: YAMLMetadata{Name: "test_tracking_plan"},
					Spec: TrackingPlanSpec{
						ID: "test_plan_001", DisplayName: "Test Tracking Plan", Description: "A test tracking plan for validation",
						Rules: []EventRule{
							{
								Type: "track", ID: "product_viewed_rule",
								Event: EventRuleRef{Ref: "#/events/product_viewed", AllowUnplanned: false},
								Properties: []PropertyRuleRef{
									{Ref: "#/properties/product_id", Required: true},
									{Ref: "#/properties/product_name", Required: false},
								},
							},
						},
					},
				}
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				yamlString := string(yamlData)
				assert.Contains(t, yamlString, "version: rudder/v0.1")
				assert.Contains(t, yamlString, "kind: tracking-plan")
				assert.Contains(t, yamlString, "test_tracking_plan")
				assert.Contains(t, yamlString, "Test Tracking Plan")
				assert.Contains(t, yamlString, "product_viewed_rule")
				assert.Contains(t, yamlString, "#/events/product_viewed")
				assert.Contains(t, yamlString, "allow_unplanned: false")
			},
		},
		{
			category: "TrackingPlan",
			name:     "UnmarshalFromYAML",
			unmarshal: func(t *testing.T, yamlData string) interface{} {
				data := `
version: rudder/v0.1
kind: tracking-plan
metadata:
  name: ecommerce_tracking_plan
spec:
  id: ecommerce_plan_v1
  display_name: E-commerce Tracking Plan
  description: Tracking plan for e-commerce events
  rules:
    - type: track
      id: purchase_rule
      event:
        $ref: "#/events/order_completed"
        allow_unplanned: true
        identity_section: user
      properties:
        - $ref: "#/properties/order_id"
          required: true
        - $ref: "#/properties/total_amount"
          required: true
        - $ref: "#/properties/currency"
          required: false
    - type: identify
      id: user_identify_rule
      event:
        $ref: "#/events/user_identify"
        allow_unplanned: false
      properties:
        - $ref: "#/properties/user_email"
          required: true`

				var trackingPlanYAML TrackingPlanYAML
				err := yaml.Unmarshal([]byte(data), &trackingPlanYAML)
				require.NoError(t, err)
				return trackingPlanYAML
			},
			validate: func(t *testing.T, original interface{}, yamlData []byte) {
				trackingPlanYAML := original.(TrackingPlanYAML)
				assert.Equal(t, "rudder/v0.1", trackingPlanYAML.Version)
				assert.Equal(t, "tracking-plan", trackingPlanYAML.Kind)
				assert.Equal(t, "ecommerce_tracking_plan", trackingPlanYAML.Metadata.Name)
				assert.Equal(t, "ecommerce_plan_v1", trackingPlanYAML.Spec.ID)
				assert.Equal(t, "E-commerce Tracking Plan", trackingPlanYAML.Spec.DisplayName)
				assert.Equal(t, "Tracking plan for e-commerce events", trackingPlanYAML.Spec.Description)
				assert.Len(t, trackingPlanYAML.Spec.Rules, 2)

				rule1 := trackingPlanYAML.Spec.Rules[0]
				assert.Equal(t, "track", rule1.Type)
				assert.Equal(t, "purchase_rule", rule1.ID)
				assert.Equal(t, "#/events/order_completed", rule1.Event.Ref)
				assert.True(t, rule1.Event.AllowUnplanned)
				assert.Equal(t, "user", rule1.Event.IdentitySection)
				assert.Len(t, rule1.Properties, 3)

				assert.Equal(t, "#/properties/order_id", rule1.Properties[0].Ref)
				assert.True(t, rule1.Properties[0].Required)
				assert.Equal(t, "#/properties/currency", rule1.Properties[2].Ref)
				assert.False(t, rule1.Properties[2].Required)

				rule2 := trackingPlanYAML.Spec.Rules[1]
				assert.Equal(t, "identify", rule2.Type)
				assert.Equal(t, "user_identify_rule", rule2.ID)
				assert.Equal(t, "#/events/user_identify", rule2.Event.Ref)
				assert.False(t, rule2.Event.AllowUnplanned)
				assert.Empty(t, rule2.Event.IdentitySection)
				assert.Len(t, rule2.Properties, 1)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.category+"/"+c.name, func(t *testing.T) {
			t.Parallel()

			if c.unmarshal != nil {
				// Unmarshal test
				result := c.unmarshal(t, "")
				c.validate(t, result, nil)
			} else {
				// Marshal test
				original := c.setup()
				yamlData, err := yaml.Marshal(original)
				require.NoError(t, err)
				assert.NotEmpty(t, yamlData)
				c.validate(t, original, yamlData)
			}
		})
	}
}

func TestYAMLMetadata(t *testing.T) {
	t.Parallel()

	t.Run("SimpleMetadata", func(t *testing.T) {
		t.Parallel()

		metadata := YAMLMetadata{Name: "test_metadata"}
		yamlData, err := yaml.Marshal(metadata)
		require.NoError(t, err)

		var restored YAMLMetadata
		err = yaml.Unmarshal(yamlData, &restored)
		require.NoError(t, err)

		assert.Equal(t, metadata.Name, restored.Name)
	})
}

func TestPropertyConfig_EdgeCases(t *testing.T) {
	t.Parallel()

	configTests := []struct {
		name   string
		config PropertyConfig
	}{
		{"EmptyConfig", PropertyConfig{}},
		{"StringConfig", PropertyConfig{MinLength: func() *int { i := 1; return &i }(), MaxLength: func() *int { i := 100; return &i }(), Pattern: "^[a-zA-Z0-9]+$"}},
		{"NumberConfig", PropertyConfig{Minimum: func() *float64 { f := 0.0; return &f }(), Maximum: func() *float64 { f := 1000.0; return &f }()}},
		{"ArrayConfig", PropertyConfig{MinItems: func() *int { i := 1; return &i }(), MaxItems: func() *int { i := 10; return &i }(), UniqueItems: func() *bool { b := true; return &b }()}},
	}

	for _, test := range configTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			yamlData, err := yaml.Marshal(test.config)
			require.NoError(t, err)
			assert.NotEmpty(t, yamlData)
			var restored PropertyConfig
			err = yaml.Unmarshal(yamlData, &restored)
			require.NoError(t, err)
		})
	}
}

func TestCustomTypeConfig_EdgeCases(t *testing.T) {
	t.Parallel()

	customTypeTests := []struct {
		name   string
		config CustomTypeConfig
	}{
		{"EmptyConfig", CustomTypeConfig{}},
		{"ArrayTypeConfig", CustomTypeConfig{MinItems: func() *int { i := 1; return &i }(), MaxItems: func() *int { i := 50; return &i }(), UniqueItems: func() *bool { b := true; return &b }(), ItemTypes: []string{"#/custom-types/item_type"}}},
		{"ObjectTypeConfig", CustomTypeConfig{MinItems: func() *int { i := 1; return &i }(), MaxItems: func() *int { i := 10; return &i }()}},
	}

	for _, test := range customTypeTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			yamlData, err := yaml.Marshal(test.config)
			require.NoError(t, err)
			assert.NotEmpty(t, yamlData)
			var restored CustomTypeConfig
			err = yaml.Unmarshal(yamlData, &restored)
			require.NoError(t, err)
		})
	}
}
