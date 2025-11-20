package differ

import (
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
)

var (
	isNilTypes = []reflect.Kind{reflect.Map, reflect.Slice, reflect.Pointer}
)

// Diff represents the differences between two resource graphs
type Diff struct {
	// NewResources contains URNs of resources that will be created (exist in target but not in source, and have no ImportMetadata)
	NewResources []string
	// ImportableResources contains URNs of resources that will be imported (exist in target but not in source, and have ImportMetadata)
	ImportableResources []string
	// UpdatedResources contains URNs of resources that exist in both graphs but have different data
	UpdatedResources map[string]ResourceDiff
	// RemovedResources contains URNs of resources that exist in source but not in target
	RemovedResources []string
	// UnmodifiedResources contains URNs of resources that exist in both graphs with identical data
	UnmodifiedResources []string
}

func (d *Diff) HasDiff() bool {
	return len(d.NewResources) > 0 ||
		len(d.ImportableResources) > 0 ||
		len(d.UpdatedResources) > 0 ||
		len(d.RemovedResources) > 0
}

type ResourceDiff struct {
	URN   string
	Diffs map[string]PropertyDiff
}

type PropertyDiff struct {
	Property    string
	SourceValue interface{}
	TargetValue interface{}
}

type DiffOptions struct {
	WorkspaceID string
}

// ComputeDiff computes the diff between two graphs
// It returns a Diff struct containing the new, importable, updated, removed and unmodified resources
// - New resources are resources that will be created (exist in target but not in source, without ImportMetadata)
// - Importable resources are resources that will be imported (exist in target but not in source, with ImportMetadata)
// - Updated resources are resources that exist in both but their data differ
// - Removed resources are resources that exist in the source but not in the target
// - Unmodified resources are resources that exist in both with identical data
func ComputeDiff(source *resources.Graph, target *resources.Graph, options DiffOptions) *Diff {
	newResources := []string{}
	importableResources := []string{}
	removedResources := []string{}
	updatedResources := map[string]ResourceDiff{}
	unmodifiedResources := []string{}

	// Iterate over target resources to find new and updated resources
	for urn, r := range target.Resources() {
		if sourceResource, exists := source.GetResource(urn); !exists {
			// Categorize based on ImportMetadata - mutually exclusive
			if r.ImportMetadata() != nil && r.ImportMetadata().WorkspaceId == options.WorkspaceID {
				// Resource will be imported (exists remotely)
				importableResources = append(importableResources, urn)
			} else {
				// Resource will be created (doesn't exist anywhere)
				newResources = append(newResources, urn)
			}
		} else {
			// Check if resource is updated or unmodified
			// Resources have either Data or RawData, never both
			var propertyDiffs map[string]PropertyDiff
			if r.RawData() != nil && sourceResource.RawData() != nil {
				// Compare RawData using reflection
				propertyDiffs = CompareRawData(sourceResource.RawData(), r.RawData())
			} else {
				// Compare Data (map-based)
				propertyDiffs = CompareData(sourceResource.Data(), r.Data())
			}

			if len(propertyDiffs) > 0 {
				updatedResources[urn] = ResourceDiff{URN: urn, Diffs: propertyDiffs}
			} else {
				unmodifiedResources = append(unmodifiedResources, urn)
			}
		}
	}

	// Iterate over source resources to find removed resources
	for urn := range source.Resources() {
		if _, exists := target.GetResource(urn); !exists {
			// Resource is removed if it doesn't exist in the target
			removedResources = append(removedResources, urn)
		}
	}

	return &Diff{
		NewResources:        newResources,
		ImportableResources: importableResources,
		UpdatedResources:    updatedResources,
		RemovedResources:    removedResources,
		UnmodifiedResources: unmodifiedResources,
	}
}

// CompareRawData compares two RawData objects using reflection and returns the differences
func CompareRawData(raw1, raw2 any) map[string]PropertyDiff {
	diffs := make(map[string]PropertyDiff)

	val1 := reflect.ValueOf(raw1)
	val2 := reflect.ValueOf(raw2)

	// Dereference pointers
	if val1.Kind() == reflect.Ptr {
		val1 = val1.Elem()
		val2 = val2.Elem()
	}

	// Compare struct fields
	if val1.Kind() == reflect.Struct {
		typ := val1.Type()
		for i := 0; i < val1.NumField(); i++ {
			field := typ.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			field1 := val1.Field(i)
			field2 := val2.Field(i)

			// Check if field has diff tag
			tag, hasDiffTag := field.Tag.Lookup("diff")

			// Get interface values, handling invalid/zero values
			var val1Interface, val2Interface any
			if field1.IsValid() && field1.CanInterface() {
				val1Interface = field1.Interface()
			}
			if field2.IsValid() && field2.CanInterface() {
				val2Interface = field2.Interface()
			}

			if !compareRawValues(val1Interface, val2Interface) {
				if hasDiffTag && tag != "" {
					// Field has diff tag - use it as the field name
					// Try to find nested diffs first
					nestedDiffs := compareRawDataNested(val1Interface, val2Interface, tag)
					if len(nestedDiffs) > 0 {
						// Found nested diffs with diff tags
						for k, v := range nestedDiffs {
							diffs[k] = v
						}
					} else {
						// No nested diffs found
						// Only report this field if it's a leaf value (not a struct/pointer to struct)
						// This avoids reporting parent fields when all nested fields are nil
						// PropertyRef is treated as a leaf value even though it's a struct
						val1Type := reflect.TypeOf(val1Interface)
						val2Type := reflect.TypeOf(val2Interface)
						isPropertyRef := false
						if val1Type != nil {
							if val1Type == reflect.TypeOf(&resources.PropertyRef{}) || val1Type == reflect.TypeOf(resources.PropertyRef{}) {
								isPropertyRef = true
							}
						}
						if val2Type != nil {
							if val2Type == reflect.TypeOf(&resources.PropertyRef{}) || val2Type == reflect.TypeOf(resources.PropertyRef{}) {
								isPropertyRef = true
							}
						}

						isStruct := false
						if !isPropertyRef {
							if val1Type != nil && (val1Type.Kind() == reflect.Struct || (val1Type.Kind() == reflect.Ptr && val1Type.Elem().Kind() == reflect.Struct)) {
								isStruct = true
							}
							if val2Type != nil && (val2Type.Kind() == reflect.Struct || (val2Type.Kind() == reflect.Ptr && val2Type.Elem().Kind() == reflect.Struct)) {
								isStruct = true
							}
						}

						if !isStruct || isPropertyRef {
							// This is a leaf field (primitive, slice, map, PropertyRef, etc.), report it
							diffs[tag] = PropertyDiff{
								Property:    tag,
								SourceValue: val1Interface,
								TargetValue: val2Interface,
							}
						}
						// If it's a struct with no nested diffs, don't report it
						// (all fields inside are nil or equal)
					}
				} else {
					// No diff tag - only traverse into nested structures
					nestedDiffs := compareRawDataNested(val1Interface, val2Interface, "")
					for k, v := range nestedDiffs {
						diffs[k] = v
					}
				}
			}
		}
		return diffs
	}

	// For non-struct types, use direct comparison
	if !compareRawValues(raw1, raw2) {
		diffs["_raw_data"] = PropertyDiff{Property: "_raw_data", SourceValue: raw1, TargetValue: raw2}
	}

	return diffs
}

// compareRawDataNested traverses nested structures/slices to find diffs in fields with diff tags
// prefix is used to build the full path to nested fields (e.g., "parent.child")
func compareRawDataNested(v1, v2 any, prefix string) map[string]PropertyDiff {
	diffs := make(map[string]PropertyDiff)

	if isNil(v1) && isNil(v2) {
		return diffs
	}

	// If one is nil and the other is not, collect all nested fields from the non-nil value
	if isNil(v1) || isNil(v2) {
		nonNilVal := v2
		nilVal := v1
		if isNil(v2) {
			nonNilVal = v1
			nilVal = v2
		}
		// Collect all nested diff-tagged fields from the non-nil value
		nestedDiffs := collectAllNestedFields(nonNilVal, nilVal, prefix)
		return nestedDiffs
	}

	val1 := reflect.ValueOf(v1)
	val2 := reflect.ValueOf(v2)

	// Dereference pointers
	if val1.Kind() == reflect.Ptr {
		if val1.IsNil() || val2.IsNil() {
			// One is nil, collect nested fields from non-nil
			nonNilVal := v2
			nilVal := v1
			if val2.IsNil() {
				nonNilVal = v1
				nilVal = v2
			}
			return collectAllNestedFields(nonNilVal, nilVal, prefix)
		}
		val1 = val1.Elem()
		val2 = val2.Elem()
	}

	switch val1.Kind() {
	case reflect.Struct:
		// Recursively compare struct fields
		typ := val1.Type()
		for i := 0; i < val1.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}

			field1 := val1.Field(i)
			field2 := val2.Field(i)

			// Check if field has diff tag
			tag, hasDiffTag := field.Tag.Lookup("diff")

			// Get interface values, handling invalid/zero values
			var val1Interface, val2Interface any
			if field1.IsValid() && field1.CanInterface() {
				val1Interface = field1.Interface()
			}
			if field2.IsValid() && field2.CanInterface() {
				val2Interface = field2.Interface()
			}

			if !compareRawValues(val1Interface, val2Interface) {
				if hasDiffTag && tag != "" {
					// Build the full field path
					fullPath := tag
					if prefix != "" {
						fullPath = prefix + "." + tag
					}

					// Try to find nested diffs first
					nestedDiffs := compareRawDataNested(val1Interface, val2Interface, fullPath)
					if len(nestedDiffs) > 0 {
						// Found nested diffs with diff tags
						for k, v := range nestedDiffs {
							diffs[k] = v
						}
					} else {
						// No nested diffs found
						// Only report this field if it's a leaf value (not a struct/pointer to struct)
						// This avoids reporting parent fields when all nested fields are nil
						// PropertyRef is treated as a leaf value even though it's a struct
						val1Type := reflect.TypeOf(val1Interface)
						val2Type := reflect.TypeOf(val2Interface)
						isPropertyRef := false
						if val1Type != nil {
							if val1Type == reflect.TypeOf(&resources.PropertyRef{}) || val1Type == reflect.TypeOf(resources.PropertyRef{}) {
								isPropertyRef = true
							}
						}
						if val2Type != nil {
							if val2Type == reflect.TypeOf(&resources.PropertyRef{}) || val2Type == reflect.TypeOf(resources.PropertyRef{}) {
								isPropertyRef = true
							}
						}

						isStruct := false
						if !isPropertyRef {
							if val1Type != nil && (val1Type.Kind() == reflect.Struct || (val1Type.Kind() == reflect.Ptr && val1Type.Elem().Kind() == reflect.Struct)) {
								isStruct = true
							}
							if val2Type != nil && (val2Type.Kind() == reflect.Struct || (val2Type.Kind() == reflect.Ptr && val2Type.Elem().Kind() == reflect.Struct)) {
								isStruct = true
							}
						}

						if !isStruct || isPropertyRef {
							// This is a leaf field (primitive, slice, map, PropertyRef, etc.), report it
							diffs[fullPath] = PropertyDiff{
								Property:    fullPath,
								SourceValue: val1Interface,
								TargetValue: val2Interface,
							}
						}
						// If it's a struct with no nested diffs, don't report it
						// (all fields inside are nil or equal)
					}
				} else {
					// No diff tag - continue traversing but don't add to prefix
					nestedDiffs := compareRawDataNested(val1Interface, val2Interface, prefix)
					for k, v := range nestedDiffs {
						diffs[k] = v
					}
				}
			}
		}

	case reflect.Slice, reflect.Array:
		// For slices/arrays, traverse each element
		if val1.Len() == val2.Len() {
			for i := 0; i < val1.Len(); i++ {
				nestedDiffs := compareRawDataNested(val1.Index(i).Interface(), val2.Index(i).Interface(), prefix)
				for k, v := range nestedDiffs {
					diffs[k] = v
				}
			}
		}

	case reflect.Map:
		// For maps, traverse each value
		for _, key := range val1.MapKeys() {
			val2Item := val2.MapIndex(key)
			if val2Item.IsValid() {
				nestedDiffs := compareRawDataNested(val1.MapIndex(key).Interface(), val2Item.Interface(), prefix)
				for k, v := range nestedDiffs {
					diffs[k] = v
				}
			}
		}
	}

	return diffs
}

// collectAllNestedFields collects all nested fields with diff tags from a structure
// This is used when comparing nil vs non-nil to report all nested field differences
func collectAllNestedFields(nonNilVal, nilVal any, prefix string) map[string]PropertyDiff {
	diffs := make(map[string]PropertyDiff)

	if isNil(nonNilVal) {
		return diffs
	}

	val := reflect.ValueOf(nonNilVal)

	// Dereference pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return diffs
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return diffs
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		// Check if field has diff tag
		tag, hasDiffTag := field.Tag.Lookup("diff")

		if hasDiffTag && tag != "" {
			// Build the full field path
			fullPath := tag
			if prefix != "" {
				fullPath = prefix + "." + tag
			}

			fieldVal := val.Field(i)
			var fieldInterface any
			if fieldVal.IsValid() && fieldVal.CanInterface() {
				fieldInterface = fieldVal.Interface()
			}

			// Recursively collect nested fields
			if !isNil(fieldInterface) {
				nestedDiffs := collectAllNestedFields(fieldInterface, nil, fullPath)
				if len(nestedDiffs) > 0 {
					// Found nested fields with tags
					for k, v := range nestedDiffs {
						diffs[k] = v
					}
				} else {
					// This is a leaf field with diff tag and has a non-nil value
					diffs[fullPath] = PropertyDiff{
						Property:    fullPath,
						SourceValue: nilVal,
						TargetValue: fieldInterface,
					}
				}
			}
			// Skip nil fields - don't report nil to nil
		} else {
			// No diff tag, traverse into nested structures
			fieldVal := val.Field(i)
			var fieldInterface any
			if fieldVal.IsValid() && fieldVal.CanInterface() {
				fieldInterface = fieldVal.Interface()
			}

			if !isNil(fieldInterface) {
				nestedDiffs := collectAllNestedFields(fieldInterface, nil, prefix)
				for k, v := range nestedDiffs {
					diffs[k] = v
				}
			}
		}
	}

	return diffs
}

// compareRawValues compares two values deeply, handling PropertyRefs and other special types
func compareRawValues(v1, v2 any) bool {
	if isNil(v1) && isNil(v2) {
		return true
	}

	if isNil(v1) || isNil(v2) {
		return false
	}

	// Handle PropertyRef specially
	if ref1, ok := v1.(*resources.PropertyRef); ok {
		if ref2, ok := v2.(*resources.PropertyRef); ok {
			return comparePropertyRefs(ref1, ref2)
		}
		return false
	}
	if ref1, ok := v1.(resources.PropertyRef); ok {
		if ref2, ok := v2.(resources.PropertyRef); ok {
			return comparePropertyRefs(&ref1, &ref2)
		}
		return false
	}

	val1 := reflect.ValueOf(v1)
	val2 := reflect.ValueOf(v2)

	if val1.Type() != val2.Type() {
		return false
	}

	switch val1.Kind() {
	case reflect.Ptr:
		if val1.IsNil() && val2.IsNil() {
			return true
		}
		if val1.IsNil() || val2.IsNil() {
			return false
		}
		return compareRawValues(val1.Elem().Interface(), val2.Elem().Interface())

	case reflect.Struct:
		typ := val1.Type()
		for i := 0; i < val1.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			if !compareRawValues(val1.Field(i).Interface(), val2.Field(i).Interface()) {
				return false
			}
		}
		return true

	case reflect.Slice, reflect.Array:
		if val1.Len() != val2.Len() {
			return false
		}
		for i := 0; i < val1.Len(); i++ {
			if !compareRawValues(val1.Index(i).Interface(), val2.Index(i).Interface()) {
				return false
			}
		}
		return true

	case reflect.Map:
		if val1.Len() != val2.Len() {
			return false
		}
		for _, key := range val1.MapKeys() {
			val1Item := val1.MapIndex(key)
			val2Item := val2.MapIndex(key)
			if !val2Item.IsValid() {
				return false
			}
			if !compareRawValues(val1Item.Interface(), val2Item.Interface()) {
				return false
			}
		}
		return true

	case reflect.Func:
		// Functions cannot be compared, so we consider them equal
		return true

	default:
		return reflect.DeepEqual(v1, v2)
	}
}

// compareData compares the data of two resources and returns the differences
func CompareData(r1, r2 resources.ResourceData) map[string]PropertyDiff {
	diffs := make(map[string]PropertyDiff)

	// Helper function to compare values recursively
	var compareValues func(key string, v1, v2 interface{})
	compareValues = func(key string, v1, v2 interface{}) {

		if isNil(v1) && isNil(v2) {
			return
		}

		newV1, ok := rewriteCompatibleType(v1)
		if ok {
			v1 = newV1
		}

		if reflect.TypeOf(v1) != reflect.TypeOf(v2) {
			diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
			return
		}

		// If v1 and v2 are pointers, compare the dereferenced values
		if reflect.TypeOf(v1).Kind() == reflect.Pointer {
			if isNil(v1) || isNil(v2) {
				diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
				return
			}
			compareValues(key, reflect.ValueOf(v1).Elem().Interface(), reflect.ValueOf(v2).Elem().Interface())
			return
		}

		switch v1Typed := v1.(type) {

		case *resources.PropertyRef:
			v2Typed := v2.(*resources.PropertyRef)
			if !comparePropertyRefs(v1Typed, v2Typed) {
				diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
			}
		case resources.PropertyRef:
			v2Typed := v2.(resources.PropertyRef)
			if !comparePropertyRefs(&v1Typed, &v2Typed) {
				diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
			}
		case []map[string]interface{}:
			v2Typed := v2.([]map[string]interface{})
			if len(v1Typed) != len(v2Typed) {
				diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
				return
			}
			for i := range v1Typed {
				compareValues(key, v1Typed[i], v2Typed[i])
			}

		case map[string]interface{}:
			v2Typed := v2.(map[string]interface{})
			subDiffs := CompareData(v1Typed, v2Typed)
			if len(subDiffs) > 0 {
				diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
			}
		case []interface{}:
			v2Typed := v2.([]interface{})
			if len(v1Typed) != len(v2Typed) {
				diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
				return
			}
			for i := range v1Typed {
				compareValues(key, v1Typed[i], v2Typed[i])
			}
		default:
			if v1 != v2 {
				diffs[key] = PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2}
			}
		}
	}

	// Iterate over properties in r1 to find differences
	for key, value1 := range r1 {
		if value2, exists := r2[key]; !exists {
			diffs[key] = PropertyDiff{Property: key, SourceValue: value1, TargetValue: nil}
		} else {
			compareValues(key, value1, value2)
		}
	}

	// Iterate over properties in r2 to find properties that are not in r1
	for key, value2 := range r2 {
		if _, exists := r1[key]; !exists {
			diffs[key] = PropertyDiff{Property: key, SourceValue: nil, TargetValue: value2}
		}
	}

	return diffs
}

// rewrite []interface{} ->  map[string]interface{} if possible
// and return back the response.
func rewriteCompatibleType(input interface{}) (interface{}, bool) {

	if _, ok := input.([]interface{}); !ok {
		return nil, false
	}

	slice := input.([]interface{})

	output := make([]map[string]interface{}, len(slice))
	for i, item := range slice {
		if _, ok := item.(map[string]interface{}); !ok {
			return nil, false
		}
		output[i] = item.(map[string]interface{})
	}

	return output, true
}

func isNil(val interface{}) bool {
	if val == nil {
		return true
	}

	if lo.Contains(isNilTypes, reflect.ValueOf(val).Kind()) {
		return reflect.ValueOf(val).IsNil()
	}

	return false
}

// comparePropertyRefs compares two PropertyRef objects by their comparable fields
// (excludes the Resolve function field which cannot be compared)
func comparePropertyRefs(r1, r2 *resources.PropertyRef) bool {
	if r1 == nil && r2 == nil {
		return true
	}
	if r1 == nil || r2 == nil {
		return false
	}
	return r1.URN == r2.URN &&
		r1.Property == r2.Property &&
		r1.IsResolved == r2.IsResolved &&
		r1.Value == r2.Value
}
