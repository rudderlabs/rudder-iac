package plan

// ExtractAllCustomTypes extracts all custom types from a tracking plan
// Returns a map of custom type name to CustomType
func (tp *TrackingPlan) ExtractAllCustomTypes() map[string]*CustomType {
	customTypes := make(map[string]*CustomType)

	// Extract all custom types from event rules
	for _, rule := range tp.Rules {
		extractCustomTypesFromSchema(&rule.Schema, customTypes)
	}

	// Extract properties from within custom type schemas
	for _, customType := range customTypes {
		if !customType.IsPrimitive() && customType.Schema != nil {
			extractCustomTypesFromSchema(customType.Schema, customTypes)
		}
	}

	return customTypes
}

// ExtractAllProperties extracts all properties from a tracking plan
// Returns a map of property name to Property
func (tp *TrackingPlan) ExtractAllProperties() map[string]*Property {
	properties := make(map[string]*Property)

	// Extract all properties from event rules
	for _, rule := range tp.Rules {
		extractPropertiesFromSchema(&rule.Schema, properties)
	}

	// Extract all custom types first to then process their schemas
	customTypes := tp.ExtractAllCustomTypes()

	// Extract properties from within custom type schemas
	for _, customType := range customTypes {
		if !customType.IsPrimitive() && customType.Schema != nil {
			extractPropertiesFromSchema(customType.Schema, properties)
		}
	}

	return properties
}

// extractCustomTypesFromSchema recursively extracts custom types from an ObjectSchema
// The customTypes map is modified in place to accumulate results
func extractCustomTypesFromSchema(schema *ObjectSchema, customTypes map[string]*CustomType) {
	for _, propSchema := range schema.Properties {
		for _, t := range propSchema.Property.Type {
			if IsCustomType(t) {
				customType := AsCustomType(t)
				if customType != nil {
					customTypes[customType.Name] = customType

					// Recursively process referenced by object custom types
					if customType.Schema != nil {
						extractCustomTypesFromSchema(customType.Schema, customTypes)
					}
				}
			}
		}
	}
}

// extractPropertiesFromSchema recursively extracts all properties from an ObjectSchema
// The properties map is modified in place to accumulate results
func extractPropertiesFromSchema(schema *ObjectSchema, properties map[string]*Property) {
	for _, propSchema := range schema.Properties {
		properties[propSchema.Property.Name] = &propSchema.Property

		// Recursively process nested schemas
		if propSchema.Schema != nil {
			extractPropertiesFromSchema(propSchema.Schema, properties)
		}
	}
}
