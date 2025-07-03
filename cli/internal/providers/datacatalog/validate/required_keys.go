package validate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/localcatalog"
)

type RequiredKeysValidator struct {
}

var validTypes = []string{
	"string", "number", "integer", "boolean", "null", "array", "object",
}

var validFormatValues = []string{
	"datetime",
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

func (rk *RequiredKeysValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	log.Info("validating required keys on the entities in catalog")

	var errors []ValidationError

	for group, props := range dc.Properties {
		for _, prop := range props {
			reference := fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID)

			if prop.Name == "" || prop.Type == "" || prop.LocalID == "" {
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

			// Validate array type properties with custom type references in itemTypes
			if prop.Type == "array" && prop.Config != nil {
				if itemTypes, ok := prop.Config["itemTypes"]; ok {
					itemTypesArray, ok := itemTypes.([]any)
					if !ok {
						errors = append(errors, ValidationError{
							error:     fmt.Errorf("itemTypes must be an array"),
							Reference: reference,
						})
						continue
					}

					for idx, itemType := range itemTypesArray {
						val, ok := itemType.(string)
						if !ok {
							errors = append(errors, ValidationError{
								error:     fmt.Errorf("itemTypes at idx: %d must be string value", idx),
								Reference: reference,
							})
							continue
						}

						if catalog.CustomTypeRegex.Match([]byte(val)) {
							if len(itemTypesArray) != 1 {
								errors = append(errors, ValidationError{
									error:     fmt.Errorf("itemTypes containing custom type at idx: %d cannot be paired with other types", idx),
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

		if tp.Name == "" {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("name field is mandatory on the tracking plan"),
				Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
			})
		}

		for _, rule := range tp.Rules {
			if rule.LocalID == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id field is mandatory on the rules in tracking plan"),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

			if rule.Event == nil && rule.Includes == nil {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("event or includes section within the rules: %s in tracking plan are mandatory", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

			if rule.Event != nil && rule.Includes != nil {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("event and includes both section within the rules: %s in tracking plan are not allowed", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

			if rule.Event == nil && len(rule.Properties) > 0 {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("properties without events in rule: %s are not allowed", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
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
			if !slices.Contains(validTypes, customType.Type) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("invalid data type, acceptable values are: %s", strings.Join(validTypes, ", ")),
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

	// Check minLength is a number
	if minLength, ok := config["minLength"]; ok {
		if !isInteger(minLength) {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("minLength must be a number"),
				Reference: reference,
			})
		}
	}

	// Check maxLength is a number
	if maxLength, ok := config["maxLength"]; ok {
		if !isInteger(maxLength) {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("maxLength must be a number"),
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
	numericFields := []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"}
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

	// Check itemTypes is an array with a single item
	if itemTypes, ok := config["itemTypes"]; ok {
		itemTypesArray, ok := itemTypes.([]any)
		if !ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("itemTypes must be an array"),
				Reference: reference,
			})
		}

		for idx, itemType := range itemTypesArray {
			val, ok := itemType.(string)
			if !ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("itemTypes at idx: %d must be string value", idx),
					Reference: reference,
				})

				continue
			}

			if catalog.CustomTypeRegex.Match([]byte(val)) {
				if len(itemTypesArray) != 1 {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("itemTypes containing custom type at idx: %d cannot be paired with other types", idx),
						Reference: reference,
					})
				}

				continue
			}

			if !slices.Contains(validTypes, val) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("itemTypes at idx: %d is invalid, valid type values are: %s", idx, strings.Join(validTypes, ",")),
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
