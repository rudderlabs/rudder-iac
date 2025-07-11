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

			// Sort property references alphabetically by Ref
			if len(propertyRefs) > 0 {
				sort.Slice(propertyRefs, func(i, j int) bool {
					return propertyRefs[i].Ref < propertyRefs[j].Ref
				})
				customType.Properties = propertyRefs
			}
		}

		customTypes = append(customTypes, customType)
	}

	// No post-processing needed since custom types now have deterministic IDs based on content

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

// deduplicateCustomTypes removes duplicates based on final YAML structure
func (sa *SchemaAnalyzer) deduplicateCustomTypes(customTypes []yamlModels.CustomTypeDefinition) ([]yamlModels.CustomTypeDefinition, map[string]string) {
	seen := make(map[string]yamlModels.CustomTypeDefinition)
	remappingTable := make(map[string]string) // maps removed ID -> kept ID
	var result []yamlModels.CustomTypeDefinition

	for _, customType := range customTypes {
		// Generate signature based on type and content
		var signature string
		if customType.Type == "array" {
			// For arrays, use type + itemTypes
			itemTypes := ""
			if customType.Config != nil && len(customType.Config.ItemTypes) > 0 {
				sortedItemTypes := make([]string, len(customType.Config.ItemTypes))
				copy(sortedItemTypes, customType.Config.ItemTypes)
				sort.Strings(sortedItemTypes)
				itemTypes = strings.Join(sortedItemTypes, "|")
			}
			signature = fmt.Sprintf("array:%s", itemTypes)
		} else {
			// For objects, use type + properties
			var propertyRefs []string
			for _, prop := range customType.Properties {
				propertyRefs = append(propertyRefs, fmt.Sprintf("%s:%t", prop.Ref, prop.Required))
			}
			sort.Strings(propertyRefs)
			signature = fmt.Sprintf("object:%s", strings.Join(propertyRefs, "|"))
		}

		// Check if we've seen this signature before
		if existing, exists := seen[signature]; exists {
			// Keep the one with shorter/simpler name or ID (preference for existing)
			if len(customType.Name) < len(existing.Name) ||
				(len(customType.Name) == len(existing.Name) && customType.ID < existing.ID) {
				// customType becomes the kept one, existing becomes the removed one
				remappingTable[existing.ID] = customType.ID
				seen[signature] = customType
			} else {
				// existing stays as the kept one, customType becomes the removed one
				remappingTable[customType.ID] = existing.ID
			}
		} else {
			seen[signature] = customType
		}
	}

	// Convert back to slice, maintaining sorted order
	var signatures []string
	for sig := range seen {
		signatures = append(signatures, sig)
	}
	sort.Strings(signatures)

	for _, sig := range signatures {
		result = append(result, seen[sig])
	}

	// Debug: Check if any custom types were removed without being in remapping table
	originalIDs := make(map[string]bool)
	for _, ct := range customTypes {
		originalIDs[ct.ID] = true
	}

	resultIDs := make(map[string]bool)
	for _, ct := range result {
		resultIDs[ct.ID] = true
	}

	fmt.Printf("DEDUPLICATION ANALYSIS:\n")
	fmt.Printf("Original count: %d, Final count: %d\n", len(customTypes), len(result))

	for originalID := range originalIDs {
		if !resultIDs[originalID] && remappingTable[originalID] == "" {
			fmt.Printf("WARNING: Custom type %s was removed without mapping!\n", originalID)
		}
	}

	return result, remappingTable
}

// updatePropertyReferences updates property references to use correct custom type IDs after deduplication
func (sa *SchemaAnalyzer) updatePropertyReferences(remappingTable map[string]string) {
	updateCount := 0

	// Update all property type references that point to removed custom types
	for propertyKey, property := range sa.Properties {
		// Check if this property has a custom type reference that needs to be updated
		if strings.Contains(property.Type, "#/custom-types/extracted_custom_types/") {
			// Extract the custom type ID from the reference
			for removedID, newID := range remappingTable {
				oldRef := fmt.Sprintf("#/custom-types/extracted_custom_types/%s", removedID)
				newRef := fmt.Sprintf("#/custom-types/extracted_custom_types/%s", newID)

				// Update the property type reference if it matches
				if property.Type == oldRef {
					fmt.Printf("Updating property %s: %s → %s\n", propertyKey, oldRef, newRef)
					property.Type = newRef
					updateCount++
					break
				}
			}
		}
	}

	fmt.Printf("Updated %d property references\n", updateCount)

	// Also need to update the CustomTypes map itself to remove the duplicated entries
	for removedID := range remappingTable {
		delete(sa.CustomTypes, removedID)
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

	// Sort property references alphabetically by Ref
	sort.Slice(properties, func(i, j int) bool {
		return properties[i].Ref < properties[j].Ref
	})

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
