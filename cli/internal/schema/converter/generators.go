package converter

import (
	"fmt"
	"sort"
	"strings"

	internalModels "github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
)

// GenerateEventsYAML creates the events.yaml structure
func (sa *SchemaAnalyzer) GenerateEventsYAML() *yamlModels.EventsYAML {
	var events []yamlModels.EventDefinition

	// Sort events by ID for consistent output
	var eventKeys []string
	for key := range sa.Events {
		eventKeys = append(eventKeys, key)
	}
	sort.Strings(eventKeys)

	for _, key := range eventKeys {
		eventInfo := sa.Events[key]
		event := yamlModels.EventDefinition{
			ID:          eventInfo.ID,
			Name:        eventInfo.Name,
			EventType:   eventInfo.EventType,
			Description: eventInfo.Description,
		}
		events = append(events, event)
	}

	return &yamlModels.EventsYAML{
		Version: "rudder/0.1",
		Kind:    "events",
		Metadata: yamlModels.YAMLMetadata{
			Name: "extracted_events",
		},
		Spec: yamlModels.EventsSpec{
			Events: events,
		},
	}
}

// GeneratePropertiesYAML creates the properties.yaml structure
func (sa *SchemaAnalyzer) GeneratePropertiesYAML() *yamlModels.PropertiesYAML {
	var properties []yamlModels.PropertyDefinition

	// Sort properties by path for consistent output
	var propertyKeys []string
	for key := range sa.Properties {
		propertyKeys = append(propertyKeys, key)
	}
	sort.Strings(propertyKeys)

	for _, key := range propertyKeys {
		propertyInfo := sa.Properties[key]
		property := yamlModels.PropertyDefinition{
			ID:          propertyInfo.ID,
			Name:        propertyInfo.Name,
			Type:        propertyInfo.Type,
			Description: propertyInfo.Description,
		}
		properties = append(properties, property)
	}

	return &yamlModels.PropertiesYAML{
		Version: "rudder/v0.1",
		Kind:    "properties",
		Metadata: yamlModels.YAMLMetadata{
			Name: "extracted_properties",
		},
		Spec: yamlModels.PropertiesSpec{
			Properties: properties,
		},
	}
}

// GenerateCustomTypesYAML creates the custom-types.yaml structure
func (sa *SchemaAnalyzer) GenerateCustomTypesYAML() *yamlModels.CustomTypesYAML {
	var customTypes []yamlModels.CustomTypeDefinition

	// Sort custom types by ID for consistent output
	var typeKeys []string
	for key := range sa.CustomTypes {
		typeKeys = append(typeKeys, key)
	}
	sort.Strings(typeKeys)

	for _, key := range typeKeys {
		typeInfo := sa.CustomTypes[key]

		customType := yamlModels.CustomTypeDefinition{
			ID:          typeInfo.ID,
			Name:        typeInfo.Name,
			Type:        typeInfo.Type,
			Description: typeInfo.Description,
		}

		// Add configuration based on type
		if typeInfo.Type == "array" {
			customType.Config = &yamlModels.CustomTypeConfig{
				ItemTypes: []string{typeInfo.ArrayItemType},
			}
		}

		// Add property references for object types
		if typeInfo.Type == "object" && len(typeInfo.Structure) > 0 {
			var propertyRefs []yamlModels.PropertyRef

			// Sort structure keys for consistent output
			var structKeys []string
			for structKey := range typeInfo.Structure {
				structKeys = append(structKeys, structKey)
			}
			sort.Strings(structKeys)

			for _, structKey := range structKeys {
				// Find the corresponding property
				propertyID := findPropertyForStructField(sa, structKey, typeInfo)
				if propertyID != "" {
					propertyRef := yamlModels.PropertyRef{
						Ref:      fmt.Sprintf("#/properties/extracted_properties/%s", propertyID),
						Required: false, // Default to optional
					}
					propertyRefs = append(propertyRefs, propertyRef)
				}
			}

			if len(propertyRefs) > 0 {
				customType.Properties = propertyRefs
			}
		}

		customTypes = append(customTypes, customType)
	}

	return &yamlModels.CustomTypesYAML{
		Version: "rudder/v0.1",
		Kind:    "custom-types",
		Metadata: yamlModels.YAMLMetadata{
			Name: "extracted_custom_types",
		},
		Spec: yamlModels.CustomTypesSpec{
			Types: customTypes,
		},
	}
}

// GenerateTrackingPlansYAML creates tracking plan YAML structures grouped by writeKey
func (sa *SchemaAnalyzer) GenerateTrackingPlansYAML(schemas []internalModels.Schema) map[string]*yamlModels.TrackingPlanYAML {
	// Group schemas by writeKey
	writeKeyGroups := make(map[string][]internalModels.Schema)
	for _, schema := range schemas {
		writeKey := schema.WriteKey
		writeKeyGroups[writeKey] = append(writeKeyGroups[writeKey], schema)
	}

	trackingPlans := make(map[string]*yamlModels.TrackingPlanYAML)

	// Keep track of used rule IDs globally to ensure uniqueness
	usedRuleIDs := make(map[string]bool)

	for writeKey, groupSchemas := range writeKeyGroups {
		planID := fmt.Sprintf("tracking_plan_%s", sanitizeID(writeKey))
		displayName := fmt.Sprintf("Tracking Plan for WriteKey %s", writeKey)

		var rules []yamlModels.EventRule

		// Create rules for each schema in the group
		for i, schema := range groupSchemas {
			// Find the actual event ID that was created for this schema
			eventID := sa.findEventIDForSchema(schema)

			// Skip if no event was created for this schema (e.g., due to filtering)
			if eventID == "" {
				continue
			}

			// Generate globally unique rule ID
			ruleID := generateUniqueRuleID(writeKey, eventID, i, usedRuleIDs)

			// Get properties for this schema
			properties := sa.extractPropertiesForSchema(schema)

			rule := yamlModels.EventRule{
				Type: "event_rule",
				ID:   ruleID,
				Event: yamlModels.EventRuleRef{
					Ref:            fmt.Sprintf("#/events/extracted_events/%s", eventID),
					AllowUnplanned: false,
				},
				Properties: properties,
			}

			rules = append(rules, rule)
		}

		trackingPlan := &yamlModels.TrackingPlanYAML{
			Version: "rudder/0.1",
			Kind:    "tp",
			Metadata: yamlModels.YAMLMetadata{
				Name: planID,
			},
			Spec: yamlModels.TrackingPlanSpec{
				ID:          planID,
				DisplayName: displayName,
				Description: fmt.Sprintf("Auto-generated tracking plan for writeKey: %s", writeKey),
				Rules:       rules,
			},
		}

		trackingPlans[writeKey] = trackingPlan
	}

	return trackingPlans
}

// extractPropertiesForSchema extracts property references for a specific schema
func (sa *SchemaAnalyzer) extractPropertiesForSchema(schema internalModels.Schema) []yamlModels.PropertyRuleRef {
	var properties []yamlModels.PropertyRuleRef

	// Recursively collect all property paths from the schema
	propertyPaths := sa.collectPropertyPaths(schema.Schema, "")

	// Sort for consistent output
	sort.Strings(propertyPaths)

	for _, path := range propertyPaths {
		// Find the property in our analyzed properties
		propertyKey := sa.findPropertyKeyForPath(path)
		if propertyKey != "" {
			propertyInfo := sa.Properties[propertyKey]
			propertyRef := yamlModels.PropertyRuleRef{
				Ref:      fmt.Sprintf("#/properties/extracted_properties/%s", propertyInfo.ID),
				Required: sa.isPropertyRequired(path), // Basic heuristic
			}
			properties = append(properties, propertyRef)
		}
	}

	return properties
}

// collectPropertyPaths recursively collects all property paths from a schema
func (sa *SchemaAnalyzer) collectPropertyPaths(obj interface{}, path string) []string {
	var paths []string

	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			fieldPath := buildPath(path, key)
			paths = append(paths, fieldPath)

			// Recursively collect from nested structures
			nestedPaths := sa.collectPropertyPaths(value, fieldPath)
			paths = append(paths, nestedPaths...)
		}
	case []interface{}:
		// For arrays, we don't add the array itself as a path,
		// but we analyze the items
		if len(v) > 0 {
			nestedPaths := sa.collectPropertyPaths(v[0], path+"_item")
			paths = append(paths, nestedPaths...)
		}
	}

	return paths
}

// findPropertyKeyForPath finds the property key for a given path
func (sa *SchemaAnalyzer) findPropertyKeyForPath(path string) string {
	// Try to find exact match first
	for key, property := range sa.Properties {
		if property.Path == path {
			return key
		}
	}

	// Try to find by property name
	propertyName := extractPropertyName(path)
	for key, property := range sa.Properties {
		if property.Name == propertyName {
			return key
		}
	}

	return ""
}

// isPropertyRequired determines if a property should be marked as required
func (sa *SchemaAnalyzer) isPropertyRequired(path string) bool {
	// Basic heuristic: common required fields
	requiredFields := map[string]bool{
		"userId":      true,
		"anonymousId": true,
		"event":       true,
		"messageId":   true,
		"type":        true,
	}

	propertyName := extractPropertyName(path)
	return requiredFields[propertyName]
}

// findPropertyForStructField finds the property ID for a structure field
func findPropertyForStructField(sa *SchemaAnalyzer, fieldName string, typeInfo *CustomTypeInfo) string {
	// Look for properties that match the field name in the context of the custom type path
	for _, property := range sa.Properties {
		// Check if the property path ends with the field name
		// This handles nested paths like "properties.user.name" matching field "name"
		pathParts := strings.Split(property.Path, ".")
		if len(pathParts) > 0 && pathParts[len(pathParts)-1] == fieldName {
			return property.ID
		}

		// Also check if the property name matches
		if property.Name == fieldName {
			return property.ID
		}
	}
	return ""
}

// generateUniqueRuleID generates globally unique rule IDs across all tracking plans
func generateUniqueRuleID(writeKey, eventID string, index int, usedRuleIDs map[string]bool) string {
	sanitizedWriteKey := sanitizeID(writeKey)

	// Try the basic pattern first
	baseRuleID := fmt.Sprintf("%s_%s_rule", sanitizedWriteKey, eventID)

	// If it's not used, return it
	if !usedRuleIDs[baseRuleID] {
		usedRuleIDs[baseRuleID] = true
		return baseRuleID
	}

	// If the base ID is already used, add index to make it unique
	var finalRuleID string
	counter := index
	for {
		finalRuleID = fmt.Sprintf("%s_%s_rule_%d", sanitizedWriteKey, eventID, counter)
		if !usedRuleIDs[finalRuleID] {
			usedRuleIDs[finalRuleID] = true
			break
		}
		counter++
	}

	return finalRuleID
}

// findEventIDForSchema finds the event ID that was created for a given schema
func (sa *SchemaAnalyzer) findEventIDForSchema(schema internalModels.Schema) string {
	// Look through all events to find the one that matches this schema
	for _, event := range sa.Events {
		if event.Original.EventIdentifier == schema.EventIdentifier &&
			event.Original.WriteKey == schema.WriteKey {
			return event.ID
		}
	}
	return ""
}
