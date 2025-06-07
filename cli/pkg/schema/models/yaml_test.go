package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestEventsYAML_Marshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalToYAML", func(t *testing.T) {
		t.Parallel()

		eventsYAML := EventsYAML{
			Version: "rudder/v0.1",
			Kind:    "events",
			Metadata: YAMLMetadata{
				Name: "test_events",
			},
			Spec: EventsSpec{
				Events: []EventDefinition{
					{
						ID:          "product_viewed",
						Name:        "Product Viewed",
						EventType:   "track",
						Description: "User viewed a product",
					},
					{
						ID:        "user_signup",
						EventType: "identify",
					},
				},
			},
		}

		yamlData, err := yaml.Marshal(eventsYAML)
		require.NoError(t, err)
		assert.NotEmpty(t, yamlData)

		yamlString := string(yamlData)
		assert.Contains(t, yamlString, "version: rudder/v0.1")
		assert.Contains(t, yamlString, "kind: events")
		assert.Contains(t, yamlString, "name: test_events")
		assert.Contains(t, yamlString, "product_viewed")
		assert.Contains(t, yamlString, "Product Viewed")
		assert.Contains(t, yamlString, "User viewed a product")
	})

	t.Run("UnmarshalFromYAML", func(t *testing.T) {
		t.Parallel()

		yamlData := `
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
      event_type: page
`

		var eventsYAML EventsYAML
		err := yaml.Unmarshal([]byte(yamlData), &eventsYAML)
		require.NoError(t, err)

		assert.Equal(t, "rudder/v0.1", eventsYAML.Version)
		assert.Equal(t, "events", eventsYAML.Kind)
		assert.Equal(t, "extracted_events", eventsYAML.Metadata.Name)
		assert.Len(t, eventsYAML.Spec.Events, 2)

		// Check first event
		event1 := eventsYAML.Spec.Events[0]
		assert.Equal(t, "order_completed", event1.ID)
		assert.Equal(t, "Order Completed", event1.Name)
		assert.Equal(t, "track", event1.EventType)
		assert.Equal(t, "User completed an order", event1.Description)

		// Check second event
		event2 := eventsYAML.Spec.Events[1]
		assert.Equal(t, "page_viewed", event2.ID)
		assert.Equal(t, "", event2.Name) // Optional field
		assert.Equal(t, "page", event2.EventType)
		assert.Equal(t, "", event2.Description) // Optional field
	})

	t.Run("RoundTripMarshaling", func(t *testing.T) {
		t.Parallel()

		original := EventsYAML{
			Version: "rudder/v0.1",
			Kind:    "events",
			Metadata: YAMLMetadata{
				Name: "round_trip_test",
			},
			Spec: EventsSpec{
				Events: []EventDefinition{
					{
						ID:          "test_event_1",
						Name:        "Test Event 1",
						EventType:   "track",
						Description: "First test event",
					},
				},
			},
		}

		// Marshal to YAML
		yamlData, err := yaml.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back to struct
		var restored EventsYAML
		err = yaml.Unmarshal(yamlData, &restored)
		require.NoError(t, err)

		assert.Equal(t, original.Version, restored.Version)
		assert.Equal(t, original.Kind, restored.Kind)
		assert.Equal(t, original.Metadata.Name, restored.Metadata.Name)
		assert.Len(t, restored.Spec.Events, 1)
		assert.Equal(t, original.Spec.Events[0], restored.Spec.Events[0])
	})
}

func TestPropertiesYAML_Marshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalToYAML", func(t *testing.T) {
		t.Parallel()

		minLen := 3
		maxLen := 50
		min := 0.0
		max := 100.0

		propertiesYAML := PropertiesYAML{
			Version: "rudder/v0.1",
			Kind:    "properties",
			Metadata: YAMLMetadata{
				Name: "test_properties",
			},
			Spec: PropertiesSpec{
				Properties: []PropertyDefinition{
					{
						ID:          "user_email",
						Name:        "User Email",
						Type:        "string",
						Description: "User's email address",
						PropConfig: &PropertyConfig{
							MinLength: &minLen,
							MaxLength: &maxLen,
							Pattern:   "^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$",
						},
					},
					{
						ID:   "user_age",
						Name: "User Age",
						Type: "number",
						PropConfig: &PropertyConfig{
							Minimum: &min,
							Maximum: &max,
						},
					},
				},
			},
		}

		yamlData, err := yaml.Marshal(propertiesYAML)
		require.NoError(t, err)
		assert.NotEmpty(t, yamlData)

		yamlString := string(yamlData)
		assert.Contains(t, yamlString, "version: rudder/v0.1")
		assert.Contains(t, yamlString, "kind: properties")
		assert.Contains(t, yamlString, "user_email")
		assert.Contains(t, yamlString, "User Email")
		assert.Contains(t, yamlString, "propConfig")
		assert.Contains(t, yamlString, "minLength: 3")
		assert.Contains(t, yamlString, "maxLength: 50")
	})

	t.Run("UnmarshalFromYAML", func(t *testing.T) {
		t.Parallel()

		yamlData := `
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
    - id: product_price
      name: Product Price
      type: number
      propConfig:
        minimum: 0
        maximum: 999999.99
    - id: categories
      name: Categories
      type: array
      propConfig:
        minItems: 1
        maxItems: 10
        uniqueItems: true
`

		var propertiesYAML PropertiesYAML
		err := yaml.Unmarshal([]byte(yamlData), &propertiesYAML)
		require.NoError(t, err)

		assert.Equal(t, "rudder/v0.1", propertiesYAML.Version)
		assert.Equal(t, "properties", propertiesYAML.Kind)
		assert.Equal(t, "extracted_properties", propertiesYAML.Metadata.Name)
		assert.Len(t, propertiesYAML.Spec.Properties, 3)

		// Check first property
		prop1 := propertiesYAML.Spec.Properties[0]
		assert.Equal(t, "product_name", prop1.ID)
		assert.Equal(t, "Product Name", prop1.Name)
		assert.Equal(t, "string", prop1.Type)
		assert.Equal(t, "Name of the product", prop1.Description)
		require.NotNil(t, prop1.PropConfig)
		assert.Equal(t, 1, *prop1.PropConfig.MinLength)
		assert.Equal(t, 100, *prop1.PropConfig.MaxLength)

		// Check third property (array type)
		prop3 := propertiesYAML.Spec.Properties[2]
		assert.Equal(t, "categories", prop3.ID)
		assert.Equal(t, "array", prop3.Type)
		require.NotNil(t, prop3.PropConfig)
		assert.Equal(t, 1, *prop3.PropConfig.MinItems)
		assert.Equal(t, 10, *prop3.PropConfig.MaxItems)
		assert.True(t, *prop3.PropConfig.UniqueItems)
	})
}

func TestCustomTypesYAML_Marshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalToYAML", func(t *testing.T) {
		t.Parallel()

		customTypesYAML := CustomTypesYAML{
			Version: "rudder/v0.1",
			Kind:    "custom-types",
			Metadata: YAMLMetadata{
				Name: "test_custom_types",
			},
			Spec: CustomTypesSpec{
				Types: []CustomTypeDefinition{
					{
						ID:          "user_profile",
						Name:        "UserProfile",
						Type:        "object",
						Description: "User profile information",
						Properties: []PropertyRef{
							{Ref: "#/properties/user_email", Required: true},
							{Ref: "#/properties/user_name", Required: true},
							{Ref: "#/properties/user_age", Required: false},
						},
					},
				},
			},
		}

		yamlData, err := yaml.Marshal(customTypesYAML)
		require.NoError(t, err)
		assert.NotEmpty(t, yamlData)

		yamlString := string(yamlData)
		assert.Contains(t, yamlString, "version: rudder/v0.1")
		assert.Contains(t, yamlString, "kind: custom-types")
		assert.Contains(t, yamlString, "user_profile")
		assert.Contains(t, yamlString, "UserProfile")
		assert.Contains(t, yamlString, "User profile information")
		assert.Contains(t, yamlString, "#/properties/user_email")
		assert.Contains(t, yamlString, "required: true")
		assert.Contains(t, yamlString, "required: false")
	})

	t.Run("UnmarshalFromYAML", func(t *testing.T) {
		t.Parallel()

		yamlData := `
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
        itemTypes: ["#/custom-types/product_type"]
`

		var customTypesYAML CustomTypesYAML
		err := yaml.Unmarshal([]byte(yamlData), &customTypesYAML)
		require.NoError(t, err)

		assert.Equal(t, "rudder/v0.1", customTypesYAML.Version)
		assert.Equal(t, "custom-types", customTypesYAML.Kind)
		assert.Equal(t, "extracted_custom_types", customTypesYAML.Metadata.Name)
		assert.Len(t, customTypesYAML.Spec.Types, 2)

		// Check first type (object)
		type1 := customTypesYAML.Spec.Types[0]
		assert.Equal(t, "address_type", type1.ID)
		assert.Equal(t, "AddressType", type1.Name)
		assert.Equal(t, "object", type1.Type)
		assert.Equal(t, "Address information", type1.Description)
		assert.Len(t, type1.Properties, 3)
		assert.True(t, type1.Properties[0].Required)
		assert.True(t, type1.Properties[1].Required)
		assert.False(t, type1.Properties[2].Required)

		// Check second type (array)
		type2 := customTypesYAML.Spec.Types[1]
		assert.Equal(t, "product_array", type2.ID)
		assert.Equal(t, "ProductArray", type2.Name)
		assert.Equal(t, "array", type2.Type)
		require.NotNil(t, type2.Config)
		assert.Equal(t, 1, *type2.Config.MinItems)
		assert.Equal(t, 100, *type2.Config.MaxItems)
		assert.Len(t, type2.Config.ItemTypes, 1)
		assert.Equal(t, "#/custom-types/product_type", type2.Config.ItemTypes[0])
	})
}

func TestTrackingPlanYAML_Marshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalToYAML", func(t *testing.T) {
		t.Parallel()

		trackingPlanYAML := TrackingPlanYAML{
			Version: "rudder/v0.1",
			Kind:    "tracking-plan",
			Metadata: YAMLMetadata{
				Name: "test_tracking_plan",
			},
			Spec: TrackingPlanSpec{
				ID:          "test_plan_001",
				DisplayName: "Test Tracking Plan",
				Description: "A test tracking plan for validation",
				Rules: []EventRule{
					{
						Type: "track",
						ID:   "product_viewed_rule",
						Event: EventRuleRef{
							Ref:            "#/events/product_viewed",
							AllowUnplanned: false,
						},
						Properties: []PropertyRuleRef{
							{Ref: "#/properties/product_id", Required: true},
							{Ref: "#/properties/product_name", Required: false},
						},
					},
				},
			},
		}

		yamlData, err := yaml.Marshal(trackingPlanYAML)
		require.NoError(t, err)
		assert.NotEmpty(t, yamlData)

		yamlString := string(yamlData)
		assert.Contains(t, yamlString, "version: rudder/v0.1")
		assert.Contains(t, yamlString, "kind: tracking-plan")
		assert.Contains(t, yamlString, "test_tracking_plan")
		assert.Contains(t, yamlString, "Test Tracking Plan")
		assert.Contains(t, yamlString, "product_viewed_rule")
		assert.Contains(t, yamlString, "#/events/product_viewed")
		assert.Contains(t, yamlString, "allow_unplanned: false")
	})

	t.Run("UnmarshalFromYAML", func(t *testing.T) {
		t.Parallel()

		yamlData := `
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
          required: true
`

		var trackingPlanYAML TrackingPlanYAML
		err := yaml.Unmarshal([]byte(yamlData), &trackingPlanYAML)
		require.NoError(t, err)

		assert.Equal(t, "rudder/v0.1", trackingPlanYAML.Version)
		assert.Equal(t, "tracking-plan", trackingPlanYAML.Kind)
		assert.Equal(t, "ecommerce_tracking_plan", trackingPlanYAML.Metadata.Name)
		assert.Equal(t, "ecommerce_plan_v1", trackingPlanYAML.Spec.ID)
		assert.Equal(t, "E-commerce Tracking Plan", trackingPlanYAML.Spec.DisplayName)
		assert.Equal(t, "Tracking plan for e-commerce events", trackingPlanYAML.Spec.Description)
		assert.Len(t, trackingPlanYAML.Spec.Rules, 2)

		// Check first rule
		rule1 := trackingPlanYAML.Spec.Rules[0]
		assert.Equal(t, "track", rule1.Type)
		assert.Equal(t, "purchase_rule", rule1.ID)
		assert.Equal(t, "#/events/order_completed", rule1.Event.Ref)
		assert.True(t, rule1.Event.AllowUnplanned)
		assert.Equal(t, "user", rule1.Event.IdentitySection)
		assert.Len(t, rule1.Properties, 3)

		// Check properties in first rule
		assert.Equal(t, "#/properties/order_id", rule1.Properties[0].Ref)
		assert.True(t, rule1.Properties[0].Required)
		assert.Equal(t, "#/properties/currency", rule1.Properties[2].Ref)
		assert.False(t, rule1.Properties[2].Required)

		// Check second rule
		rule2 := trackingPlanYAML.Spec.Rules[1]
		assert.Equal(t, "identify", rule2.Type)
		assert.Equal(t, "user_identify_rule", rule2.ID)
		assert.Equal(t, "#/events/user_identify", rule2.Event.Ref)
		assert.False(t, rule2.Event.AllowUnplanned)
		assert.Empty(t, rule2.Event.IdentitySection)
		assert.Len(t, rule2.Properties, 1)
	})
}

func TestYAMLMetadata(t *testing.T) {
	t.Parallel()

	t.Run("SimpleMetadata", func(t *testing.T) {
		t.Parallel()

		metadata := YAMLMetadata{
			Name: "test_metadata",
		}

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

	t.Run("AllConfigOptions", func(t *testing.T) {
		t.Parallel()

		minLen := 5
		maxLen := 100
		minItems := 1
		maxItems := 10
		uniqueItems := true
		min := 0.0
		max := 1000.0

		config := PropertyConfig{
			MinLength:   &minLen,
			MaxLength:   &maxLen,
			Pattern:     "^[a-zA-Z]+$",
			Enum:        []string{"red", "blue", "green"},
			MinItems:    &minItems,
			MaxItems:    &maxItems,
			UniqueItems: &uniqueItems,
			Minimum:     &min,
			Maximum:     &max,
		}

		yamlData, err := yaml.Marshal(config)
		require.NoError(t, err)

		var restored PropertyConfig
		err = yaml.Unmarshal(yamlData, &restored)
		require.NoError(t, err)

		assert.Equal(t, *config.MinLength, *restored.MinLength)
		assert.Equal(t, *config.MaxLength, *restored.MaxLength)
		assert.Equal(t, config.Pattern, restored.Pattern)
		assert.Equal(t, config.Enum, restored.Enum)
		assert.Equal(t, *config.MinItems, *restored.MinItems)
		assert.Equal(t, *config.MaxItems, *restored.MaxItems)
		assert.Equal(t, *config.UniqueItems, *restored.UniqueItems)
		assert.Equal(t, *config.Minimum, *restored.Minimum)
		assert.Equal(t, *config.Maximum, *restored.Maximum)
	})

	t.Run("NilValues", func(t *testing.T) {
		t.Parallel()

		config := PropertyConfig{
			Pattern: "test-pattern",
			Enum:    []string{"option1"},
		}

		yamlData, err := yaml.Marshal(config)
		require.NoError(t, err)

		var restored PropertyConfig
		err = yaml.Unmarshal(yamlData, &restored)
		require.NoError(t, err)

		assert.Nil(t, restored.MinLength)
		assert.Nil(t, restored.MaxLength)
		assert.Equal(t, config.Pattern, restored.Pattern)
		assert.Equal(t, config.Enum, restored.Enum)
		assert.Nil(t, restored.MinItems)
		assert.Nil(t, restored.MaxItems)
		assert.Nil(t, restored.UniqueItems)
		assert.Nil(t, restored.Minimum)
		assert.Nil(t, restored.Maximum)
	})
}

func TestCustomTypeConfig_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("AllConfigOptions", func(t *testing.T) {
		t.Parallel()

		minLen := 3
		maxLen := 50
		minItems := 1
		maxItems := 20
		uniqueItems := true
		min := 10.5
		max := 99.9
		exclusiveMin := 0.0
		exclusiveMax := 100.0
		multipleOf := 2.5

		config := CustomTypeConfig{
			MinLength:    &minLen,
			MaxLength:    &maxLen,
			Pattern:      "^\\d+$",
			Format:       "email",
			Enum:         []string{"value1", "value2", "value3"},
			Minimum:      &min,
			Maximum:      &max,
			ExclusiveMin: &exclusiveMin,
			ExclusiveMax: &exclusiveMax,
			MultipleOf:   &multipleOf,
			ItemTypes:    []string{"string", "number"},
			MinItems:     &minItems,
			MaxItems:     &maxItems,
			UniqueItems:  &uniqueItems,
		}

		yamlData, err := yaml.Marshal(config)
		require.NoError(t, err)

		var restored CustomTypeConfig
		err = yaml.Unmarshal(yamlData, &restored)
		require.NoError(t, err)

		assert.Equal(t, *config.MinLength, *restored.MinLength)
		assert.Equal(t, *config.MaxLength, *restored.MaxLength)
		assert.Equal(t, config.Pattern, restored.Pattern)
		assert.Equal(t, config.Format, restored.Format)
		assert.Equal(t, config.Enum, restored.Enum)
		assert.Equal(t, *config.Minimum, *restored.Minimum)
		assert.Equal(t, *config.Maximum, *restored.Maximum)
		assert.Equal(t, *config.ExclusiveMin, *restored.ExclusiveMin)
		assert.Equal(t, *config.ExclusiveMax, *restored.ExclusiveMax)
		assert.Equal(t, *config.MultipleOf, *restored.MultipleOf)
		assert.Equal(t, config.ItemTypes, restored.ItemTypes)
		assert.Equal(t, *config.MinItems, *restored.MinItems)
		assert.Equal(t, *config.MaxItems, *restored.MaxItems)
		assert.Equal(t, *config.UniqueItems, *restored.UniqueItems)
	})
}
