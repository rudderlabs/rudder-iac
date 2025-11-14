package plan

// ExtractAllCustomTypes extracts all custom types from a tracking plan
// Returns a map of custom type name to CustomType
func (tp *TrackingPlan) ExtractAllCustomTypes() map[string]*CustomType {
	customTypes := make(map[string]*CustomType)

	// Extract all custom types from event rules
	for _, rule := range tp.Rules {
		extractCustomTypesFromSchema(&rule.Schema, customTypes)
		// Extract from event rule variants
		extractCustomTypesFromVariants(rule.Variants, customTypes)
	}

	// Use a worklist approach to handle newly discovered custom types
	processedTypes := make(map[string]bool)
	for len(processedTypes) < len(customTypes) {
		for name, customType := range customTypes {
			if processedTypes[name] {
				continue
			}
			processedTypes[name] = true

			if !customType.IsPrimitive() {
				extractCustomTypesFromSchema(customType.Schema, customTypes)
			}
			// Extract from custom type variants
			extractCustomTypesFromVariants(customType.Variants, customTypes)

			// Extract from custom type item types (for array custom types)
			if customType.Type == PrimitiveTypeArray && customType.ItemType != nil {
				if IsCustomType(customType.ItemType) {
					itemCustomType := AsCustomType(customType.ItemType)
					if itemCustomType != nil {
						customTypes[itemCustomType.Name] = itemCustomType
					}
				}
			}
		}
	}

	return customTypes
}

func extractCustomTypesFromCustomType(customType *CustomType, customTypes map[string]*CustomType) {
	if !customType.IsPrimitive() {
		extractCustomTypesFromSchema(customType.Schema, customTypes)
	}
	extractCustomTypesFromVariants(customType.Variants, customTypes)

	if customType.Type == PrimitiveTypeArray && customType.ItemType != nil {
		if IsCustomType(customType.ItemType) {
			itemCustomType := AsCustomType(customType.ItemType)
			if itemCustomType != nil {
				customTypes[itemCustomType.Name] = itemCustomType
				extractCustomTypesFromCustomType(itemCustomType, customTypes)
			}
		}
	}
}

// ExtractAllProperties extracts all properties from a tracking plan
// Returns a map of property name to Property
func (tp *TrackingPlan) ExtractAllProperties() map[string]*Property {
	properties := make(map[string]*Property)

	// Extract all properties from event rules
	for _, rule := range tp.Rules {
		extractPropertiesFromSchema(&rule.Schema, properties)
		// Extract from event rule variants
		extractPropertiesFromVariants(rule.Variants, properties)
	}

	// Extract all custom types first to then process their schemas
	customTypes := tp.ExtractAllCustomTypes()

	// Extract properties from within custom type schemas and variants
	for _, customType := range customTypes {
		if !customType.IsPrimitive() && customType.Schema != nil {
			extractPropertiesFromSchema(customType.Schema, properties)
		}
		// Extract from custom type variants
		extractPropertiesFromVariants(customType.Variants, properties)
	}

	return properties
}

// extractCustomTypesFromSchema recursively extracts custom types from an ObjectSchema
// The customTypes map is modified in place to accumulate results
func extractCustomTypesFromSchema(schema *ObjectSchema, customTypes map[string]*CustomType) {
	for _, propSchema := range schema.Properties {
		for _, t := range propSchema.Property.Types {
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

		// Also extract custom types from array item types
		for _, itemType := range propSchema.Property.ItemTypes {
			if IsCustomType(itemType) {
				customType := AsCustomType(itemType)
				if customType != nil {
					customTypes[customType.Name] = customType

					// Recursively process referenced custom types
					if customType.Schema != nil {
						extractCustomTypesFromSchema(customType.Schema, customTypes)
					}
				}
			}
		}

		// Recursively process nested property schemas
		if propSchema.Schema != nil {
			extractCustomTypesFromSchema(propSchema.Schema, customTypes)
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

// extractCustomTypesFromVariants extracts custom types from variant case schemas
// The customTypes map is modified in place to accumulate results
func extractCustomTypesFromVariants(variants []Variant, customTypes map[string]*CustomType) {
	for _, variant := range variants {
		// Extract from each case schema
		for _, variantCase := range variant.Cases {
			extractCustomTypesFromSchema(&variantCase.Schema, customTypes)
		}
		// Extract from default schema if present
		if variant.DefaultSchema != nil {
			extractCustomTypesFromSchema(variant.DefaultSchema, customTypes)
		}
	}
}

// extractPropertiesFromVariants extracts properties from variant case schemas
// The properties map is modified in place to accumulate results
func extractPropertiesFromVariants(variants []Variant, properties map[string]*Property) {
	for _, variant := range variants {
		// Extract from each case schema
		for _, variantCase := range variant.Cases {
			extractPropertiesFromSchema(&variantCase.Schema, properties)
		}
		// Extract from default schema if present
		if variant.DefaultSchema != nil {
			extractPropertiesFromSchema(variant.DefaultSchema, properties)
		}
	}
}
