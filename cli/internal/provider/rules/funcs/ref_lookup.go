package funcs

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ValidateReferences walks a struct (or slice of structs) via reflection,
// finds fields tagged with pattern=legacy_*_ref, extracts the ref value,
// and checks if the referenced resource exists in the graph.
//
// By the time semantic validation runs, legacy refs (#/properties/group/id)
// have been transformed to URN format (#property:id) by transformReferencesInSpec.
// The URN format embeds the resource kind directly, so no tag-to-type mapping is needed.
//
// Returns only failed lookups — refs that don't resolve to an existing resource.
func ValidateReferences(spec any, graph *resources.Graph) []rules.ValidationResult {
	val := reflect.ValueOf(spec)

	// Dereference pointer if needed
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return walkStruct(val, val.Type(), "", graph)
	case reflect.Slice:
		return walkSlice(val, "", graph)
	default:
		return nil
	}
}

// walkStruct recursively walks a struct's fields, checking ref-tagged fields against the graph.
func walkStruct(val reflect.Value, typ reflect.Type, basePath string, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// join the base path and the JSON field name
		// to get the full JSON pointer path for the field
		fieldPath := joinRefPath(
			basePath,
			jsonTagName(field),
		)

		validateTag := field.Tag.Get("validate")

		var (
			actualVal  = fieldVal
			actualKind = fieldVal.Kind()
		)

		if actualKind == reflect.Pointer {
			if fieldVal.IsNil() {
				continue
			}
			actualVal = fieldVal.Elem()
			actualKind = actualVal.Kind()
		}

		if validateTag != "" && actualKind == reflect.String {
			patternName := extractLegacyPattern(validateTag)
			if patternName == "" {
				// return early if there is no legacy pattern
				// present on the field
				// TODO we would need to update it with the
				// new pattern name
				continue
			}

			refValue := actualVal.String()
			if !strings.HasPrefix(refValue, "#") {
				// skip empty values and non-reference strings (e.g., primitive types
				// like "string" on fields tagged with combined patterns)
				continue
			}

			if result, failed := checkRefInGraph(refValue, fieldPath, graph); failed {
				results = append(results, result)
			}
		}

		// recurse into struct fields
		if actualKind == reflect.Struct {
			results = append(results, walkStruct(actualVal, actualVal.Type(), fieldPath, graph)...)
		}

		// iterate slice fields
		if actualKind == reflect.Slice {
			results = append(results, walkSlice(actualVal, fieldPath, graph)...)
		}
	}

	return results
}

// walkSlice iterates over slice elements, recursing into struct elements.
func walkSlice(val reflect.Value, basePath string, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for j := 0; j < val.Len(); j++ {
		elemPath := fmt.Sprintf("%s/%d", basePath, j)
		elem := val.Index(j)

		// Dereference pointer elements
		if elem.Kind() == reflect.Pointer {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}

		if elem.Kind() == reflect.Struct {
			results = append(results, walkStruct(elem, elem.Type(), elemPath, graph)...)
		}
	}

	return results
}

// checkRefInGraph validates a single reference value against the graph.
// The ref is expected in URN format (#<kind>:<id>) since legacy refs are
// transformed before semantic validation runs.
// Returns a ValidationResult and true if the reference failed lookup.
func checkRefInGraph(refValue, fieldPath string, graph *resources.Graph) (rules.ValidationResult, bool) {
	resourceType, localID, err := ParseURNRef(refValue)
	if err != nil {
		return rules.ValidationResult{
			Reference: fieldPath,
			Message:   fmt.Sprintf("failed to parse reference '%s': %v", refValue, err),
		}, true
	}

	urn := resources.URN(localID, resourceType)
	if _, exists := graph.GetResource(urn); !exists {
		return rules.ValidationResult{
			Reference: fieldPath,
			Message:   fmt.Sprintf("referenced %s '%s' not found in resource graph", resourceType, localID),
		}, true
	}

	return rules.ValidationResult{}, false
}

// extractLegacyPattern extracts the legacy pattern name from a validate tag string.
// Input:  "required,pattern=legacy_property_ref"
// Output: "legacy_property_ref"
// Returns "" if no legacy pattern is found.
func extractLegacyPattern(validateTag string) string {
	for part := range strings.SplitSeq(validateTag, ",") {
		if strings.HasPrefix(part, "pattern=legacy_") {
			return strings.TrimPrefix(part, "pattern=")
		}
	}
	return ""
}

// ParseURNRef extracts the resource type and local ID from a URN-format reference.
// Input:  "#property:user_id"
// Output: "property", "user_id", nil
func ParseURNRef(ref string) (string, string, error) {
	trimmed := strings.TrimPrefix(ref, "#")
	resourceType, localID, ok := strings.Cut(trimmed, ":")
	if !ok || resourceType == "" || localID == "" {
		return "", "", fmt.Errorf("expected URN format '#<kind>:<id>', got '%s'", ref)
	}
	return resourceType, localID, nil
}

// jsonTagName extracts the JSON field name from a struct field's json tag.
// Falls back to the struct field name if no json tag is present.
func jsonTagName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return field.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// joinRefPath joins a base path and a segment into a JSON pointer path.
func joinRefPath(base, segment string) string {
	if base == "" {
		return "/" + segment
	}
	return base + "/" + segment
}
