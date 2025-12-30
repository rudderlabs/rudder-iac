package differ

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"
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
			var sData map[string]interface{}
			var tData map[string]interface{}

			if sourceResource.RawData() != nil && r.RawData() != nil {
				_ = mapstructure.Decode(sourceResource.RawData(), &sData)
				_ = mapstructure.Decode(r.RawData(), &tData)

			} else {
				sData = sourceResource.Data()
				tData = r.Data()
			}

			// Check if resource is updated or unmodified
			propertyDiffs := CompareData(sData, tData)
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
