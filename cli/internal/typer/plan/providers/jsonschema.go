package providers

import (
	"context"
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
		rule, err := parseTrackingPlanEventSchema(&ev)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}

	tp := &plan.TrackingPlan{
		Name:  apitp.Name,
		Rules: rules,
		Metadata: plan.PlanMetadata{
			TrackingPlanID:      apitp.ID,
			TrackingPlanVersion: apitp.Version,
			URL:                 fmt.Sprintf("https://app.rudderstack.com/trackingPlans/%s", apitp.ID),
		},
	}

	return tp, nil
}

func parseEventType(s string) (plan.EventType, error) {
	switch s {
	case "track":
		return plan.EventTypeTrack, nil
	case "identify":
		return plan.EventTypeIdentify, nil
	case "page":
		return plan.EventTypePage, nil
	case "screen":
		return plan.EventTypeScreen, nil
	case "group":
		return plan.EventTypeGroup, nil
	default:
		return "", fmt.Errorf("invalid event type: %s", s)
	}
}

func parseIdentitySection(s string) (plan.IdentitySection, error) {
	switch s {
	case "properties":
		return plan.IdentitySectionProperties, nil
	case "traits":
		return plan.IdentitySectionTraits, nil
	case "context.traits":
		return plan.IdentitySectionContextTraits, nil
	default:
		return "", fmt.Errorf("invalid identity section: %s", s)
	}
}

func parsePrimitiveType(s string) (plan.PrimitiveType, error) {
	switch s {
	case "string":
		return plan.PrimitiveTypeString, nil
	case "integer":
		return plan.PrimitiveTypeInteger, nil
	case "number":
		return plan.PrimitiveTypeNumber, nil
	case "boolean":
		return plan.PrimitiveTypeBoolean, nil
	case "array":
		return plan.PrimitiveTypeArray, nil
	case "object":
		return plan.PrimitiveTypeObject, nil
	case "null":
		return plan.PrimitiveTypeNull, nil
	default:
		return "", fmt.Errorf("invalid primitive type: %s", s)
	}
}

func parseTrackingPlanEventSchema(ev *catalog.TrackingPlanEventSchema) (*plan.EventRule, error) {
	evType, err := parseEventType(ev.EventType)
	if err != nil {
		return nil, err
	}

	event := plan.Event{
		Name:        ev.Name,
		Description: ev.Description,
		EventType:   evType,
	}

	customTypes, err := parseEventRulesDefs(ev)
	if err != nil {
		return nil, fmt.Errorf("parsing event rules definitions: %w", err)
	}

	// Handle different identity section formats
	var schemaProperties any
	var ok bool

	if ev.IdentitySection == "context.traits" {
		// For context.traits, navigate the nested structure: Properties["context"]["properties"]["traits"]
		contextObj, contextExists := ev.Rules.Properties["context"]
		if !contextExists {
			return nil, fmt.Errorf("identity section '%s' not found in properties: 'context' key missing", ev.IdentitySection)
		}

		contextMap, ok := contextObj.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("identity section '%s': 'context' must be an object", ev.IdentitySection)
		}

		contextProps, propsExists := contextMap["properties"]
		if !propsExists {
			return nil, fmt.Errorf("identity section '%s' not found in properties: 'context.properties' key missing", ev.IdentitySection)
		}

		contextPropsMap, ok := contextProps.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("identity section '%s': 'context.properties' must be an object", ev.IdentitySection)
		}

		schemaProperties, ok = contextPropsMap["traits"]
		if !ok {
			return nil, fmt.Errorf("identity section '%s' not found in properties: 'context.properties.traits' key missing", ev.IdentitySection)
		}
	} else {
		// For "properties" and "traits", use direct lookup
		schemaProperties, ok = ev.Rules.Properties[ev.IdentitySection]
		if !ok {
			return nil, fmt.Errorf("identity section '%s' not found in properties", ev.IdentitySection)
		}
	}

	schemaPropertiesMap, ok := schemaProperties.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("properties for identity section '%s' must be an object", ev.IdentitySection)
	}

	td, err := parseTypeDefinition(schemaPropertiesMap, customTypes)
	if err != nil {
		return nil, err
	}
	if (len(td.Types) != 1) || (td.Types[0] != plan.PrimitiveTypeObject) {
		return nil, fmt.Errorf("identity section '%s' must be of type 'object'", ev.IdentitySection)
	}

	section, err := parseIdentitySection(ev.IdentitySection)
	if err != nil {
		return nil, err
	}

	// Parse variants if present
	variants, err := parseVariants(schemaPropertiesMap, customTypes)
	if err != nil {
		return nil, fmt.Errorf("parsing variants: %w", err)
	}

	rule := plan.EventRule{
		Event:    event,
		Section:  section,
		Schema:   *td.Schema,
		Variants: variants,
	}

	return &rule, nil
}

func parseEventRulesDefs(ev *catalog.TrackingPlanEventSchema) (map[string]*plan.CustomType, error) {
	customTypes := make(map[string]*plan.CustomType)
	if ev.Rules.Defs != nil {
		customTypes = make(map[string]*plan.CustomType)

		// First pass: create custom types without resolving references
		for typeName := range ev.Rules.Defs {
			customTypes[typeName] = &plan.CustomType{
				Name: typeName,
			}
		}

		// Second pass: populate custom types with full definitions
		for typeName, typeDef := range ev.Rules.Defs {
			if typeDefMap, ok := typeDef.(map[string]any); ok {
				td, err := parseTypeDefinition(typeDefMap, customTypes)
				if err != nil {
					return nil, fmt.Errorf("parsing custom type '%s': %w", typeName, err)
				}

				// custom types can only have a single primitive type
				if len(td.Types) != 1 {
					return nil, fmt.Errorf("custom type '%s' must have a single type, got %d types", typeName, len(td.Types))
				}
				customTypeType := td.Types[0]
				if !plan.IsPrimitiveType(customTypeType) {
					return nil, fmt.Errorf("custom type '%s' must be a primitive type, got '%s'", typeName, customTypeType)
				}

				ct := customTypes[typeName]
				ct.Type = *plan.AsPrimitiveType(td.Types[0])
				ct.Schema = td.Schema
				ct.Config = td.Config

				// For array types, set ItemType
				if ct.Type == plan.PrimitiveTypeArray && len(td.ItemTypes) > 0 {
					ct.ItemType = td.ItemTypes[0]
				}

				// Parse variants for custom types
				variants, err := parseVariants(typeDefMap, customTypes)
				if err != nil {
					return nil, fmt.Errorf("parsing variants for custom type '%s': %w", typeName, err)
				}
				ct.Variants = variants
			} else {
				return nil, fmt.Errorf("custom type definition for '%s' must be an object", typeName)
			}
		}

	}

	return customTypes, nil
}

type typeDefinition struct {
	Types     []plan.PropertyType
	ItemTypes []plan.PropertyType
	Config    *plan.PropertyConfig
	Schema    *plan.ObjectSchema
}

func parseTypeDefinition(def map[string]any, customTypes map[string]*plan.CustomType) (*typeDefinition, error) {
	td := &typeDefinition{}
	// check for $ref first
	if ref, exists := def["$ref"]; exists {
		customType, err := resolveCustomTypeReference(ref.(string), customTypes)
		if err != nil {
			return nil, fmt.Errorf("resolving custom type reference '%v': %w", ref, err)
		}

		td.Types = []plan.PropertyType{customType}
		return td, nil
	}

	// parse enums
	if enumVal, exists := def["enum"]; exists {
		if enumSlice, ok := enumVal.([]any); ok {
			td.Config = &plan.PropertyConfig{
				Enum: enumSlice,
			}
		}
	}

	// parse type
	typeVal, exists := def["type"]
	if !exists {
		td.Types = []plan.PropertyType{}
	} else {
		switch v := typeVal.(type) {
		case string:
			pt, err := parsePrimitiveType(v)
			if err != nil {
				return nil, fmt.Errorf("parsing primitive type '%s': %w", v, err)
			}
			td.Types = []plan.PropertyType{pt}
		case []any:
			if len(v) == 0 {
				td.Types = []plan.PropertyType{}
			} else {
				for _, item := range v {
					if str, ok := item.(string); ok {
						pt, err := parsePrimitiveType(str)
						if err != nil {
							return nil, fmt.Errorf("parsing primitive type '%s': %w", str, err)
						}
						td.Types = append(td.Types, pt)
					}
				}
			}
		}
	}

	// handle nested objects
	if len(td.Types) > 0 && td.Types[0] == plan.PrimitiveTypeObject {
		objSchema := &plan.ObjectSchema{
			Properties: make(map[string]plan.PropertySchema),
		}
		td.Schema = objSchema

		// Parse additionalProperties and unevaluatedProperties fields
		// Priority: additionalProperties > unevaluatedProperties > default to true
		if additionalPropsVal, exists := def["additionalProperties"]; exists {
			if additionalProps, ok := additionalPropsVal.(bool); ok {
				objSchema.AdditionalProperties = additionalProps
			}
		} else if unevaluatedPropsVal, exists := def["unevaluatedProperties"]; exists {
			// If additionalProperties is not specified, use unevaluatedProperties
			if unevaluatedProps, ok := unevaluatedPropsVal.(bool); ok {
				objSchema.AdditionalProperties = unevaluatedProps
			}
		} else {
			objSchema.AdditionalProperties = true // default to true if neither is specified
		}

		if propertiesMap, exists := def["properties"]; exists {
			if props, ok := propertiesMap.(map[string]any); ok {
				for propName, propDef := range props {
					ptd, err := parseTypeDefinition(propDef.(map[string]any), customTypes)
					if err != nil {
						return nil, fmt.Errorf("parsing property '%s': %w", propName, err)
					}

					objSchema.Properties[propName] = plan.PropertySchema{
						Property: plan.Property{
							Name:      propName,
							Types:     ptd.Types,
							ItemTypes: ptd.ItemTypes,
							Config:    ptd.Config,
						},
						Schema: ptd.Schema,
					}
				}

				// handle required properties
				if requiredList, exists := def["required"]; exists {
					if reqs, ok := requiredList.([]any); ok {
						for _, r := range reqs {
							if rStr, ok := r.(string); ok {
								if propSchema, exists := objSchema.Properties[rStr]; exists {
									propSchema.Required = true
									objSchema.Properties[rStr] = propSchema
								}
							}
						}
					}
				}
			}
		}
	}

	// handle arrays
	if len(td.Types) > 0 && td.Types[0] == plan.PrimitiveTypeArray {
		if items, exists := def["items"]; exists {
			if itemsMap, ok := items.(map[string]any); ok {
				ref, refExists := itemsMap["$ref"]
				if refExists {
					customType, err := resolveCustomTypeReference(ref.(string), customTypes)
					if err != nil {
						return nil, fmt.Errorf("resolving array item custom type reference '%v': %w", ref, err)
					}
					td.ItemTypes = []plan.PropertyType{customType}
				} else {
					t, tExists := itemsMap["type"]
					if tExists {
						if tSlice, ok := t.([]any); ok {
							for _, item := range tSlice {
								if str, ok := item.(string); ok {
									pt, err := parsePrimitiveType(str)
									if err != nil {
										return nil, fmt.Errorf("parsing array item primitive type '%s': %w", str, err)
									}
									td.ItemTypes = append(td.ItemTypes, pt)
								}
							}
						} else if tStr, ok := t.(string); ok {
							pt, err := parsePrimitiveType(tStr)
							if err != nil {
								return nil, fmt.Errorf("parsing array item primitive type '%s': %w", tStr, err)
							}
							td.ItemTypes = []plan.PropertyType{pt}
						} else {
							return nil, fmt.Errorf("array items type must be a string or array of strings")
						}
					} else {
						return nil, fmt.Errorf("array items must have a 'type' field")
					}
				}
			}
		}
	}

	return td, nil
}

// resolveCustomTypeReference resolves a $ref reference to a custom type
func resolveCustomTypeReference(ref string, customTypes map[string]*plan.CustomType) (*plan.CustomType, error) {
	// Parse reference format: #/$defs/TypeName
	const prefix = "#/$defs/"
	if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
		return nil, fmt.Errorf("invalid $ref format: '%s', expected format: '#/$defs/TypeName'", ref)
	}

	typeName := ref[len(prefix):]
	customType, exists := customTypes[typeName]
	if !exists {
		return nil, fmt.Errorf("custom type '%s' not found in definitions", typeName)
	}

	return customType, nil
}

// parseVariants parses variant definitions from a JSON Schema allOf structure
func parseVariants(def map[string]any, customTypes map[string]*plan.CustomType) ([]plan.Variant, error) {
	allOfVal, exists := def["allOf"]
	if !exists {
		return nil, nil
	}

	allOfList, ok := allOfVal.([]any)
	if !ok {
		return nil, fmt.Errorf("allOf must be an array")
	}

	// We expect a single variant definition in allOf
	if len(allOfList) != 1 {
		return nil, fmt.Errorf("expected exactly one variant definition in allOf")
	}

	variantDef, ok := allOfList[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("variant definition must be an object")
	}

	variantAllOf, exists := variantDef["allOf"]
	if !exists {
		return nil, fmt.Errorf("variant definition must have allOf")
	}

	variantAllOfList, ok := variantAllOf.([]any)
	if !ok {
		return nil, fmt.Errorf("variant allOf must be an array")
	}

	cases, discriminator, defaultSchema, err := parseVariantCases(variantAllOfList, customTypes)
	if err != nil {
		return nil, err
	}

	variant := plan.Variant{
		Type:          "discriminator",
		Discriminator: discriminator,
		Cases:         cases,
		DefaultSchema: defaultSchema,
	}

	return []plan.Variant{variant}, nil
}

// parseVariantCases parses individual variant cases from the allOf structure
func parseVariantCases(
	allOfList []any,
	customTypes map[string]*plan.CustomType,
) ([]plan.VariantCase, string, *plan.ObjectSchema, error) {
	var cases []plan.VariantCase
	var discriminator string
	var defaultSchema *plan.ObjectSchema

	for _, item := range allOfList {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Extract if condition
		ifCondition, exists := itemMap["if"]
		if !exists {
			continue
		}

		// Extract then schema
		thenDef, exists := itemMap["then"]
		if !exists {
			continue
		}

		// Parse if condition to get discriminator and match values
		ifMap, ok := ifCondition.(map[string]any)
		if !ok {
			continue
		}

		props, exists := ifMap["properties"]
		if !exists {
			continue
		}

		propsMap, ok := props.(map[string]any)
		if !ok {
			continue
		}

		// Get discriminator property name (first property in if condition)
		for propName, propDef := range propsMap {
			if discriminator == "" {
				discriminator = propName
			}

			propDefMap, ok := propDef.(map[string]any)
			if !ok {
				continue
			}

			// Check if this is a default case (has "not")
			if _, hasNot := propDefMap["not"]; hasNot {
				// This is the default case
				thenMap, ok := thenDef.(map[string]any)
				if !ok {
					return nil, "", nil, fmt.Errorf("default case 'then' must be an object")
				}

				schema, err := parseVariantCaseSchema(thenMap, customTypes)
				if err != nil {
					return nil, "", nil, fmt.Errorf("parsing default schema: %w", err)
				}
				defaultSchema = schema
			} else if enumDef, hasEnum := propDefMap["enum"]; hasEnum {
				// This is a regular case
				enumList, ok := enumDef.([]any)
				if !ok {
					return nil, "", nil, fmt.Errorf("enum must be an array")
				}

				// Parse the then schema
				thenMap, ok := thenDef.(map[string]any)
				if !ok {
					return nil, "", nil, fmt.Errorf("case 'then' must be an object")
				}

				schema, err := parseVariantCaseSchema(thenMap, customTypes)
				if err != nil {
					return nil, "", nil, fmt.Errorf("parsing case schema: %w", err)
				}

				// Create a case for this
				variantCase := plan.VariantCase{
					Match:  enumList,
					Schema: *schema,
					// DisplayName and Description could be extracted from metadata if available
				}
				cases = append(cases, variantCase)
			}
		}
	}

	return cases, discriminator, defaultSchema, nil
}

// parseVariantCaseSchema parses an ObjectSchema from a variant case definition (if-then-else)
func parseVariantCaseSchema(
	def map[string]any,
	customTypes map[string]*plan.CustomType,
) (*plan.ObjectSchema, error) {
	schema := &plan.ObjectSchema{
		Properties: make(map[string]plan.PropertySchema),
	}

	// Parse additionalProperties and unevaluatedProperties fields
	// Priority: additionalProperties > unevaluatedProperties > default to false (for variant schemas)
	if additionalPropsVal, exists := def["additionalProperties"]; exists {
		if additionalProps, ok := additionalPropsVal.(bool); ok {
			schema.AdditionalProperties = additionalProps
		}
	} else if unevaluatedPropsVal, exists := def["unevaluatedProperties"]; exists {
		// If additionalProperties is not specified, use unevaluatedProperties
		if unevaluatedProps, ok := unevaluatedPropsVal.(bool); ok {
			schema.AdditionalProperties = unevaluatedProps
		}
	}
	// Note: No else clause here - variant schemas don't default to true

	// Parse properties
	if propertiesMap, exists := def["properties"]; exists {
		if props, ok := propertiesMap.(map[string]any); ok {
			for propName, propDef := range props {
				propDefMap, ok := propDef.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("property '%s' definition must be an object", propName)
				}

				ptd, err := parseTypeDefinition(propDefMap, customTypes)
				if err != nil {
					return nil, fmt.Errorf("parsing property '%s': %w", propName, err)
				}

				schema.Properties[propName] = plan.PropertySchema{
					Property: plan.Property{
						Name:      propName,
						Types:     ptd.Types,
						ItemTypes: ptd.ItemTypes,
						Config:    ptd.Config,
					},
					Schema: ptd.Schema,
				}
			}

			// Handle required properties
			if requiredList, exists := def["required"]; exists {
				if reqs, ok := requiredList.([]any); ok {
					for _, r := range reqs {
						if rStr, ok := r.(string); ok {
							if propSchema, exists := schema.Properties[rStr]; exists {
								propSchema.Required = true
								schema.Properties[rStr] = propSchema
							}
						}
					}
				}
			}
		}
	}

	return schema, nil
}
