package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

type JSONSchemaPlanProvider struct {
	trackingPlanID string
	client         catalog.TrackingPlanStore
}

func NewJSONSchemaPlanProvider(trackingPlanID string, client catalog.TrackingPlanStore) *JSONSchemaPlanProvider {
	return &JSONSchemaPlanProvider{
		trackingPlanID: trackingPlanID,
		client:         client,
	}
}

func (p *JSONSchemaPlanProvider) GetTrackingPlan(ctx context.Context) (*plan.TrackingPlan, error) {
	apitp, err := p.client.GetTrackingPlanWithSchemas(ctx, p.trackingPlanID)
	if err != nil {
		return nil, err
	}

	rules := make([]plan.EventRule, 0, len(apitp.Events))
	for _, ev := range apitp.Events {
		rule, err := parseEventSchema(&ev)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}

	tp := &plan.TrackingPlan{
		Name:  apitp.Name,
		Rules: rules,
	}

	// Pretty print the tracking plan for debugging
	b, err := json.MarshalIndent(tp, "", "  ")
	if err != nil {
		return nil, err
	}
	fmt.Println(string(b))

	return tp, nil
}

func parseEventSchema(ev *catalog.TrackingPlanEventSchema) (*plan.EventRule, error) {
	evType, err := plan.ParseEventType(ev.EventType)
	if err != nil {
		return nil, err
	}

	event := plan.Event{
		Name:        ev.Name,
		Description: ev.Description,
		EventType:   evType,
	}

	section, err := plan.ParseIdentitySection(ev.IdentitySection)
	if err != nil {
		return nil, err
	}

	schemaProperties, ok := ev.Rules.Properties[ev.IdentitySection]
	if !ok {
		return nil, fmt.Errorf("identity section '%s' not found in properties", ev.IdentitySection)
	}

	schemaPropertiesMap, ok := schemaProperties.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("properties for identity section '%s' must be an object", ev.IdentitySection)
	}

	schema, err := parseJSONSchemaObject(schemaPropertiesMap)
	if err != nil {
		return nil, err
	}

	rule := plan.EventRule{
		Event:   event,
		Section: section,
		Schema:  *schema,
	}

	return &rule, nil
}

func parseJSONSchemaObject(object map[string]any) (*plan.ObjectSchema, error) {
	objSchema := &plan.ObjectSchema{
		Properties: make(map[string]plan.PropertySchema),
	}

	requiredSet := make(map[string]bool)
	required, ok := object["required"]
	if ok {
		slice, ok := required.([]interface{})
		if !ok {
			return nil, fmt.Errorf("'required' field must be an array of strings, it is instead %T", required)
		}
		// Create a set of required property names for quick lookup
		for _, req := range slice {
			if reqStr, ok := req.(string); ok {
				requiredSet[reqStr] = true
			}
		}
	}

	properties, ok := object["properties"].(map[string]any)
	if !ok {
		return nil, nil
	}

	for propName, propDef := range properties {
		propSchema, err := parsePropertySchema(propName, propDef, requiredSet[propName])
		if err != nil {
			return nil, fmt.Errorf("parsing property '%s': %w", propName, err)
		}

		if propSchema != nil {
			objSchema.Properties[propName] = *propSchema
		}
	}

	return objSchema, nil
}

func parsePropertySchema(name string, propDef any, required bool) (*plan.PropertySchema, error) {
	propMap, ok := propDef.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("property definition must be an object, got %T", propDef)
	}

	// Parse type
	typeVal, exists := propMap["type"]
	if !exists {
		typeVal = "object"
	}

	typeStr, ok := typeVal.(string)
	if !ok {
		typeArr, ok := typeVal.([]any)
		if !ok || len(typeArr) == 0 {
			return nil, fmt.Errorf("property 'type' must be a string or non-empty array of strings, got %T", typeVal)
		}
		typeStr, ok = typeArr[0].(string)
		if !ok {
			return nil, fmt.Errorf("property 'type' array must contain strings, got %T", typeArr[0])
		}
	}

	propType, err := plan.ParsePrimitiveType(typeStr)
	if err != nil {
		return nil, fmt.Errorf("parsing property type '%s': %w", typeStr, err)
	}

	// Create property
	property := plan.Property{
		Name: name,
		Type: []plan.PropertyType{propType},
	}

	// Parse description if present
	if desc, exists := propMap["description"]; exists {
		if descStr, ok := desc.(string); ok {
			property.Description = descStr
		}
	}

	// Parse enum config if present
	if enumVal, exists := propMap["enum"]; exists {
		if enumSlice, ok := enumVal.([]any); ok {
			var enumStrings []string
			for _, e := range enumSlice {
				if eStr, ok := e.(string); ok {
					enumStrings = append(enumStrings, eStr)
				}
			}
			if len(enumStrings) > 0 {
				property.Config = &plan.PropertyConfig{
					Enum: enumStrings,
				}
			}
		}
	}

	// Create property schema
	propSchema := &plan.PropertySchema{
		Property: property,
		Required: required,
	}

	// Handle nested objects
	if typeStr == "object" {
		nestedObj, err := parseJSONSchemaObject(propMap)
		if err != nil {
			return nil, fmt.Errorf("parsing nested object for property '%s': %w", name, err)
		}
		propSchema.Schema = nestedObj
	}

	return propSchema, nil
}
