package property

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validatePropertySemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.PropertySpec, graph *resources.Graph) []rules.ValidationResult {
	// Generic ref validation for pattern=legacy_* tagged fields
	results := funcs.ValidateReferences(spec, graph)

	// Property-specific reference checks: Type field and Config itemTypes
	// may contain custom type refs that need graph lookup. The generic
	// walker skips these because they have no pattern=legacy_* tag.
	results = append(results, validateReferenceType(spec, graph)...)

	// (name, type, itemTypes) uniqueness across the entire resource graph
	results = append(results, validatePropertyUniqueness(spec, graph)...)

	return results
}

var validatePropertySemanticV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.PropertySpecV1, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	results = append(results, validateReferenceTypeV1(spec, graph)...)
	results = append(results, validatePropertyUniquenessV1(spec, graph)...)

	return results
}

// validateReferenceType checks custom type references in Property.Type
// and Property.Config["itemTypes"]. Both may contain URN-format refs
// (e.g., "#custom-type:Address") that need to exist in the resource graph.
func validateReferenceType(spec localcatalog.PropertySpec, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, prop := range spec.Properties {
		// Check the Type field for a custom type reference
		if strings.HasPrefix(prop.Type, "#") {
			results = append(results, checkRef(prop.Type, fmt.Sprintf("/properties/%d/type", i), graph)...)
		}

		// Check Config["itemTypes"] for custom type references
		itemTypes, ok := prop.Config["itemTypes"]
		if !ok {
			continue
		}

		items, ok := itemTypes.([]any)
		if !ok {
			continue
		}

		for j, item := range items {
			ref, ok := item.(string)
			if !ok || !strings.HasPrefix(ref, "#") {
				continue
			}
			results = append(results, checkRef(ref, fmt.Sprintf("/properties/%d/propConfig/itemTypes/%d", i, j), graph)...)
		}
	}

	return results
}

// checkRef parses a URN reference and verifies it exists in the graph.
func checkRef(ref, jsonPointer string, graph *resources.Graph) []rules.ValidationResult {
	resourceType, localID, err := funcs.ParseURNRef(ref)
	if err != nil {
		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("failed to parse reference '%s': %v", ref, err),
		}}
	}

	urn := resources.URN(localID, resourceType)
	if _, exists := graph.GetResource(urn); !exists {
		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("referenced %s '%s' not found in resource graph", resourceType, localID),
		}}
	}

	return nil
}

// validatePropertyUniqueness checks that each property's (name, type, itemTypes)
// combination is unique across the entire resource graph. If another property
// with the same combination exists, both spec files will independently report an error.
func validatePropertyUniqueness(spec localcatalog.PropertySpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.PropertyResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		config, _ := data["config"].(map[string]any)
		key := fmt.Sprintf(
			"%s|%s|%s",
			name,
			normalizeType(data["type"]),
			normalizeItemTypes(config, "item_types"),
		)
		countMap[key]++
	}

	var results []rules.ValidationResult
	for i, prop := range spec.Properties {
		key := fmt.Sprintf(
			"%s|%s|%s",
			prop.Name,
			normalizeType(prop.Type),
			normalizeItemTypes(prop.Config, "itemTypes"),
		)

		if countMap[key] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d", i),
				Message:   fmt.Sprintf("duplicate name '%s' within kind 'properties'", prop.Name),
			})
		}
	}

	return results
}

func validateReferenceTypeV1(spec localcatalog.PropertySpecV1, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, property := range spec.Properties {
		if hasCustomTypeRef(property.Type) {
			results = append(results, checkRef(property.Type, fmt.Sprintf("/properties/%d/type", i), graph)...)
		}

		if hasCustomTypeRef(property.ItemType) {
			results = append(results, checkRef(property.ItemType, fmt.Sprintf("/properties/%d/item_type", i), graph)...)
		}
	}

	return results
}

func validatePropertyUniquenessV1(spec localcatalog.PropertySpecV1, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.PropertyResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		config, _ := data["config"].(map[string]any)
		key := fmt.Sprintf(
			"%s|%s|%s",
			name,
			normalizeType(data["type"]),
			normalizeItemTypes(config, "item_types"),
		)
		countMap[key]++
	}

	var results []rules.ValidationResult
	for i, property := range spec.Properties {
		key := fmt.Sprintf(
			"%s|%s|%s",
			property.Name,
			normalizePropertyTypeV1(property),
			normalizePropertyItemTypesV1(property),
		)

		if countMap[key] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d", i),
				Message:   fmt.Sprintf("duplicate name '%s' within kind 'properties'", property.Name),
			})
		}
	}

	return results
}

func hasCustomTypeRef(typeValue string) bool {
	if !strings.HasPrefix(typeValue, "#") {
		return false
	}

	resourceType, localID, err := funcs.ParseURNRef(typeValue)
	if err != nil {
		return false
	}

	return resourceType == types.CustomTypeResourceType && localID != ""
}

func normalizePropertyTypeV1(property localcatalog.PropertyV1) string {
	if property.Type != "" {
		return normalizeType(property.Type)
	}

	if len(property.Types) == 0 {
		return ""
	}

	typesCopy := append([]string(nil), property.Types...)
	sort.Strings(typesCopy)

	return strings.Join(typesCopy, ",")
}

func normalizePropertyItemTypesV1(property localcatalog.PropertyV1) string {
	if property.ItemType != "" {
		return normalizeType(property.ItemType)
	}

	if len(property.ItemTypes) == 0 {
		return ""
	}

	itemTypesCopy := append([]string(nil), property.ItemTypes...)
	sort.Strings(itemTypesCopy)

	return strings.Join(itemTypesCopy, ",")
}

// normalizeType converts a type value (string or PropertyRef) to a
// comparable string. Comma-separated primitives are sorted so order
// doesn't matter (e.g., "string,number" == "number,string").
func normalizeType(t any) string {
	switch v := t.(type) {
	case string:
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		sort.Strings(parts)
		return strings.Join(parts, ",")
	case resources.PropertyRef:
		return "#" + v.URN
	default:
		return fmt.Sprintf("%v", t)
	}
}

// normalizeItemTypes extracts itemTypes from a config map, normalizes
// each element (handling both strings and PropertyRef), sorts them,
// and returns a stable comparable string.
func normalizeItemTypes(config map[string]any, key string) string {
	if config == nil {
		return ""
	}

	items, ok := config[key].([]any)
	if !ok {
		return ""
	}

	normalized := make([]string, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, normalizeType(item))
	}
	sort.Strings(normalized)
	return strings.Join(normalized, ",")
}

func NewPropertySemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/properties/semantic-valid",
		rules.Error,
		"property references must resolve to existing resources",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindProperties),
			validatePropertySemantic,
		),
		prules.NewSemanticPatternValidator(
			[]rules.MatchPattern{
				rules.MatchKindVersion(localcatalog.KindProperties, specs.SpecVersionV1),
			},
			validatePropertySemanticV1,
		),
	)
}
