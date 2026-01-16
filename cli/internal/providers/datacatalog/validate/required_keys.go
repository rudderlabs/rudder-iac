package validate

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

type RequiredKeysValidator struct {
}

var ValidTypes = []string{
	"string", "number", "integer", "boolean", "null", "array", "object",
}

var validFormatValues = []string{
	"date-time",
	"date",
	"time",
	"email",
	"uuid",
	"hostname",
	"ipv4",
	"ipv6",
}

// Regex for custom type name validation
var customTypeNameRegex = regexp.MustCompile(`^[A-Z][a-zA-Z0-9_-]{2,64}$`)

// Regex for category name validation
var categoryNameRegex = regexp.MustCompile(`^[A-Z_a-z][\s\w,.-]{2,64}$`)

const MAX_NESTING_DEPTH = 3

func (rk *RequiredKeysValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	log.Info("validating required keys on the entities in catalog")

	var errors []ValidationError

	for group, props := range dc.Properties {
		for _, prop := range props {
			reference := fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID)

			if prop.Name == "" || prop.LocalID == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id, name and type fields on property are mandatory"),
					Reference: reference,
				})
			}

			// Validate property name doesn't have leading or trailing whitespace
			if prop.Name != strings.TrimSpace(prop.Name) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("property name cannot have leading or trailing whitespace characters"),
					Reference: reference,
				})
			}

			if catalog.CustomTypeRegex.Match([]byte(prop.Type)) {
				if prop.Config != nil {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("property config not allowed if the type matches custom-type"),
						Reference: reference,
					})
				}

			}

			// Validate array type properties with custom type references in item_types
			if prop.Type == "array" && prop.Config != nil {
				if itemTypes, ok := prop.Config["item_types"]; ok {
					itemTypesArray, ok := itemTypes.([]any)
					if !ok {
						errors = append(errors, ValidationError{
							error:     fmt.Errorf("item_types must be an array"),
							Reference: reference,
						})
						continue
					}

					for idx, itemType := range itemTypesArray {
						val, ok := itemType.(string)
						if !ok {
							errors = append(errors, ValidationError{
								error:     fmt.Errorf("item_types at idx: %d must be string value", idx),
								Reference: reference,
							})
							continue
						}

						if catalog.CustomTypeRegex.Match([]byte(val)) {
							if len(itemTypesArray) != 1 {
								errors = append(errors, ValidationError{
									error:     fmt.Errorf("item_types containing custom type at idx: %d cannot be paired with other types", idx),
									Reference: reference,
								})
							}
						}
					}
				}
			}
		}
	}

	// Events required keys
	for group, events := range dc.Events {
		for _, event := range events {
			reference := fmt.Sprintf("#/events/%s/%s", group, event.LocalID)

			if event.LocalID == "" || event.Type == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id and event_type fields on event are mandatory"),
					Reference: reference,
				})
			}

			if event.Type == "track" {
				if event.Name == "" {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("name field is mandatory on track event"),
						Reference: reference,
					})
				}
			}
		}
	}

	// Tracking Plan required keys
	for group, tp := range dc.TrackingPlans {
		reference := fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID)

		if tp.Name == "" {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("name field is mandatory on the tracking plan"),
				Reference: reference,
			})
		}

		for _, rule := range tp.Rules {
			ruleRef := fmt.Sprintf("%s/rules/%s", reference, rule.LocalID)

			if rule.LocalID == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id field is mandatory on the rules in tracking plan"),
					Reference: reference,
				})
			}

			if rule.Event == nil && rule.Includes == nil {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("event or includes section within the rules: %s in tracking plan are mandatory", rule.LocalID),
					Reference: reference,
				})
			}

			if rule.Event != nil && rule.Includes != nil {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("event and includes both section within the rules: %s in tracking plan are not allowed", rule.LocalID),
					Reference: reference,
				})
			}

			if rule.Event == nil && len(rule.Properties) > 0 {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("properties without events in event_rule: %s are not allowed", rule.LocalID),
					Reference: reference,
				})
			}

			if len(rule.Variants) > 0 {
				errors = append(errors, rk.validateVariantsRequiredKeys(rule.Variants, ruleRef)...)
			}

			// Validate nested properties if they exist
			if rule.Event != nil && len(rule.Properties) > 0 {
				// iterate over the properties and validate each of them
				for _, prop := range rule.Properties {
					isNested := false
					// we only need to validate nested properties, there are no validations for flat properties
					if len(prop.Properties) > 0 {
						isNested = true
						errors = append(errors, rk.validateNestedProperty(prop, ruleRef, dc)...)
						errors = append(errors, rk.validateNestingDepth(prop.Properties, 1, MAX_NESTING_DEPTH, ruleRef)...)
					}

					if !isNested && prop.AdditionalProperties != nil {
						errors = append(errors, ValidationError{
							error:     fmt.Errorf("setting additional_properties is only allowed for nested properties"),
							Reference: fmt.Sprintf("%s/properties/%s", ruleRef, prop.Ref),
						})
					}
				}

			}
		}
	}

	// Custom Types required keys
	for group, customTypes := range dc.CustomTypes {
		for _, customType := range customTypes {
			reference := fmt.Sprintf("#/custom-types/%s/%s", group, customType.LocalID)

			// Check mandatory fields
			if customType.LocalID == "" || customType.Name == "" || customType.Type == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id, name and type fields on custom type are mandatory"),
					Reference: reference,
				})
				continue
			}

			// Check each property in properties has id field
			if customType.Type == "object" {

				if len(customType.Config) > 0 {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("config is not allowed on custom type of type object"),
						Reference: reference,
					})
				}

				for i, prop := range customType.Properties {
					if prop.Ref == "" {
						errors = append(errors, ValidationError{
							error:     fmt.Errorf("$ref field is mandatory for property at index %d in custom type", i),
							Reference: reference,
						})
					}
				}
			}

			// Name format validation - no need to check if name is empty as we already checked above
			if !customTypeNameRegex.MatchString(customType.Name) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("custom type name must start with a capital letter and contain only letters, numbers, underscores and dashes, and be between 2 and 65 characters long"),
					Reference: reference,
				})
			}

			// Type validation
			if !slices.Contains(ValidTypes, customType.Type) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("invalid data type, acceptable values are: %s", strings.Join(ValidTypes, ", ")),
					Reference: reference,
				})
			}

			if customType.Config != nil {
				switch customType.Type {
				case "string":
					errors = append(errors, rk.validateStringConfig(customType.Config, reference)...)
				case "number", "integer":
					errors = append(errors, rk.validateNumberConfig(customType.Config, reference, customType.Type)...)
				case "array":
					errors = append(errors, rk.validateArrayConfig(customType.Config, reference)...)
				}
			}

			if len(customType.Variants) > 0 {
				if customType.Type != "object" {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("variants are only allowed for custom type of type object"),
						Reference: reference,
					})
				} else {

					errors = append(errors, rk.validateVariantsRequiredKeys(customType.Variants, reference)...)
				}
			}
		}
	}

	// Categories required keys and format validation
	for group, categories := range dc.Categories {
		for _, category := range categories {
			reference := fmt.Sprintf("#/categories/%s/%s", group, category.LocalID)

			// Check mandatory fields
			if category.LocalID == "" || category.Name == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id and name fields on category are mandatory"),
					Reference: reference,
				})
				continue
			}

			// Validate category name doesn't have leading or trailing whitespace
			if category.Name != strings.TrimSpace(category.Name) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("category name cannot have leading or trailing whitespace characters"),
					Reference: reference,
				})
				continue
			}

			// Category name format validation using the regex from category.go
			if !categoryNameRegex.MatchString(category.Name) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("category name must start with a letter (upper/lower case) or underscore, followed by 2-64 characters including spaces, word characters, commas, periods, and hyphens"),
					Reference: reference,
				})
				continue
			}
		}
	}

	return errors
}

// validateStringConfig validates config fields for string type
func (rk *RequiredKeysValidator) validateStringConfig(config map[string]any, reference string) []ValidationError {
	var errors []ValidationError

	// Check enum is an array of strings
	if enum, ok := config["enum"]; ok {
		_, ok := enum.([]any)
		if !ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("enum must be an array"),
				Reference: reference,
			})
		}
	}

	// Check min_length is a number
	if minLength, ok := config["min_length"]; ok {
		if !isInteger(minLength) {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("min_length must be a number"),
				Reference: reference,
			})
		}
	}

	// Check max_length is a number
	if maxLength, ok := config["max_length"]; ok {
		if !isInteger(maxLength) {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("max_length must be a number"),
				Reference: reference,
			})
		}
	}

	// Check pattern is a string
	if pattern, ok := config["pattern"]; ok {
		if _, ok := pattern.(string); !ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("pattern must be a string"),
				Reference: reference,
			})
		}
	}

	// Check format is a valid value
	if format, ok := config["format"]; ok {
		formatStr, ok := format.(string)
		if !ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("format must be a string"),
				Reference: reference,
			})
		} else if !slices.Contains(validFormatValues, formatStr) {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("invalid format value, acceptable values are %s", strings.Join(validFormatValues, ", ")),
				Reference: reference,
			})
		}
	}

	return errors
}

// validateNumberConfig validates config fields for number/integer type
func (rk *RequiredKeysValidator) validateNumberConfig(config map[string]any, reference string, ctType string) []ValidationError {

	var (
		errors    []ValidationError
		typeCheck func(val any) bool = isNumber
	)

	// integer custom type has a stricter
	// check for the same items within the config
	if ctType == "integer" {
		typeCheck = isInteger
	}

	// Check enum is an array of numbers
	if enum, ok := config["enum"]; ok {
		enumArray, ok := enum.([]any)
		if !ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("enum must be an array"),
				Reference: reference,
			})
		} else {
			for i, val := range enumArray {
				if !typeCheck(val) {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("enum value at index %d must be a %s", i, ctType),
						Reference: reference,
					})
				}
			}
		}
	}

	// Check numeric fields are numbers
	numericFields := []string{"minimum", "maximum", "exclusive_minimum", "exclusive_maximum", "multiple_of"}
	for _, field := range numericFields {
		if val, ok := config[field]; ok {
			if !typeCheck(val) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("%s must be a %s", field, ctType),
					Reference: reference,
				})
			}
		}
	}

	return errors
}

// validateArrayConfig validates config fields for array type
func (rk *RequiredKeysValidator) validateArrayConfig(config map[string]any, reference string) []ValidationError {
	var errors []ValidationError

	// Check item_types is an array with a single item
	if itemTypes, ok := config["item_types"]; ok {
		itemTypesArray, ok := itemTypes.([]any)
		if !ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("item_types must be an array"),
				Reference: reference,
			})
		}

		for idx, itemType := range itemTypesArray {
			val, ok := itemType.(string)
			if !ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("item_types at idx: %d must be string value", idx),
					Reference: reference,
				})

				continue
			}

			if catalog.CustomTypeRegex.Match([]byte(val)) {
				if len(itemTypesArray) != 1 {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("item_types containing custom type at idx: %d cannot be paired with other types", idx),
						Reference: reference,
					})
				}

				continue
			}

			if !slices.Contains(ValidTypes, val) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("item_types at idx: %d is invalid, valid type values are: %s", idx, strings.Join(ValidTypes, ",")),
					Reference: reference,
				})
			}

		}
	}

	// Check numeric fields are numbers
	numericFields := []string{"minItems", "maxItems"}
	for _, field := range numericFields {
		if val, ok := config[field]; ok {
			if !isInteger(val) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("%s must be a number", field),
					Reference: reference,
				})
			}
		}
	}

	// Check uniqueItems is a boolean
	if uniqueItems, ok := config["uniqueItems"]; ok {
		if _, ok := uniqueItems.(bool); !ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("uniqueItems must be a boolean"),
				Reference: reference,
			})
		}
	}

	return errors
}

func (rk *RequiredKeysValidator) validateVariantsRequiredKeys(variants catalog.Variants, reference string) []ValidationError {
	var errors []ValidationError

	if len(variants) > 1 {
		errors = append(errors, ValidationError{
			error:     fmt.Errorf("variants array cannot have more than 1 variant (current length: %d)", len(variants)),
			Reference: reference,
		})
	}

	for i, variant := range variants {
		variantReference := fmt.Sprintf("%s/variants[%d]", reference, i)

		if variant.Type != "discriminator" {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("type field is mandatory for variant and must be 'discriminator'"),
				Reference: variantReference,
			})
		}

		if variant.Discriminator == "" {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("discriminator field is mandatory for variant"),
				Reference: variantReference,
			})
		}

		if len(variant.Cases) == 0 {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("cases array must have at least one element"),
				Reference: variantReference,
			})
		}

		for j, variantCase := range variant.Cases {
			caseReference := fmt.Sprintf("%s/cases[%d]", variantReference, j)

			if variantCase.DisplayName == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("display_name field is mandatory for variant case"),
					Reference: caseReference,
				})
			}

			if len(variantCase.Match) == 0 {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("match array must have at least one element"),
					Reference: caseReference,
				})
			}

			if len(variantCase.Properties) == 0 {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("properties array must have at least one element"),
					Reference: caseReference,
				})
			}

			for k, matchValue := range variantCase.Match {
				switch matchValue := matchValue.(type) {
				case string, bool, int:
				case float64:
					if matchValue != math.Trunc(matchValue) {
						errors = append(errors, ValidationError{
							error:     fmt.Errorf("match value at index %d must be an integer", k),
							Reference: caseReference,
						})
					}

				default:
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("match value at index %d must be string, bool or integer type (got: %T)", k, matchValue),
						Reference: caseReference,
					})
				}
			}

			for k, propRef := range variantCase.Properties {
				propReference := fmt.Sprintf("%s/properties[%d]", caseReference, k)

				if propRef.Ref == "" {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("$ref field is mandatory for property reference"),
						Reference: propReference,
					})
				}
			}
		}

		for j, propRef := range variant.Default {
			propReference := fmt.Sprintf("%s/default/properties[%d]", variantReference, j)

			if propRef.Ref == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("$ref field is mandatory for property reference"),
					Reference: propReference,
				})
			}
		}
	}

	return errors
}

func isNumber(val any) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	}

	return false
}

func isInteger(val any) bool {

	switch v := val.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return float32(int(v)) == v
	case float64:
		return float64(int64(v)) == v
	}
	return false
}

func (rk *RequiredKeysValidator) validateNestedProperty(prop *catalog.TPRuleProperty, ruleRef string, dc *catalog.DataCatalog) []ValidationError {
	if prop.Ref == "" {
		return []ValidationError{
			{
				error:     fmt.Errorf("ref field is mandatory for property %s in event_rule %s", prop.Ref, ruleRef),
				Reference: ruleRef,
			},
		}
	}

	// there is nothing to validate if the property has no nested properties
	if len(prop.Properties) == 0 {
		return nil
	}

	// validate the property reference
	matches := catalog.PropRegex.FindStringSubmatch(prop.Ref)
	if len(matches) != 3 {
		return []ValidationError{
			{
				error:     fmt.Errorf("invalid property reference format: %s in event_rule %s", prop.Ref, ruleRef),
				Reference: ruleRef,
			},
		}
	}

	propertyGroup, propertyID := matches[1], matches[2]

	// check if the property exists in the data catalog
	property := dc.Property(propertyGroup, propertyID)
	if property == nil {
		return []ValidationError{
			{
				error:     fmt.Errorf("invalid property reference: %s found in tracking plan rule: %s - referred property does not exist", prop.Ref, ruleRef),
				Reference: ruleRef,
			},
		}
	}

	// validate the property type
	allowed, err := nestedPropertiesAllowed(property.Type, property.Config)
	if !allowed {
		errs := make([]ValidationError, 0)
		errs = append(errs, ValidationError{
			error:     fmt.Errorf("nested properties are not allowed for property %s", prop.Ref),
			Reference: ruleRef,
		})

		if err != nil {
			errs = append(errs, ValidationError{
				error:     fmt.Errorf("error validating nested property %s: %w", prop.Ref, err),
				Reference: ruleRef,
			})
		}
		return errs
	}

	var errors []ValidationError
	for _, prop := range prop.Properties {
		if len(prop.Properties) > 0 {
			errors = append(errors, rk.validateNestedProperty(prop, ruleRef, dc)...)
		}
	}

	return errors
}

// validateNestingDepth validates maximum nesting depth (3 levels)
func (rk *RequiredKeysValidator) validateNestingDepth(properties []*catalog.TPRuleProperty, currentDepth int, maxDepth int, ruleRef string) []ValidationError {
	var errors []ValidationError

	if currentDepth > maxDepth {
		errors = append(errors, ValidationError{
			error:     fmt.Errorf("maximum property nesting depth of %d levels exceeded in event_rule %s", maxDepth, ruleRef),
			Reference: ruleRef,
		})
		return errors
	}

	for _, prop := range properties {
		if len(prop.Properties) > 0 {
			errors = append(errors, rk.validateNestingDepth(prop.Properties, currentDepth+1, maxDepth, ruleRef)...)
		}
	}

	return errors
}

func nestedPropertiesAllowed(propertyType string, config map[string]any) (bool, error) {
	if strings.Contains(propertyType, "object") && strings.Contains(propertyType, "array") {
		// type array and object cannot be present together
		// for a property to allow nesting
		return false, nil
	}

	if strings.Contains(propertyType, "object") {
		return true, nil
	}

	if strings.Contains(propertyType, "array") {
		itemTypes, itemTypesOk := config["item_types"]
		if !itemTypesOk {
			return true, nil
		}

		itemTypesArray, ok := itemTypes.([]any)
		if !ok {
			return false, fmt.Errorf("item_types must be an array")
		}

		found := false
		for i, itemType := range itemTypesArray {
			val, ok := itemType.(string)
			if !ok {
				return false, fmt.Errorf("item_types at index %d must be a string value", i)
			}
			if val == "object" {
				found = true
			}
		}

		if found {
			return true, nil
		}
	}

	return false, nil
}
