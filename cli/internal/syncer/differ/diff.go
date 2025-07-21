package differ

import (
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/samber/lo"
)

var (
	isNilTypes = []reflect.Kind{reflect.Map, reflect.Slice, reflect.Pointer}
)

// Diff represents the differences between two resource graphs
type Diff struct {
	// NewResources contains URNs of resources that exist in target but not in source
	NewResources []string
	// UpdatedResources contains URNs of resources that exist in both graphs but have different data
	UpdatedResources map[string]ResourceDiff
	// RemovedResources contains URNs of resources that exist in source but not in target
	RemovedResources []string
	// UnmodifiedResources contains URNs of resources that exist in both graphs with identical data
	UnmodifiedResources []string
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

// ComputeDiff computes the diff between two graphs
// It returns a Diff struct containing the new, updated, removed and unmodified resources
// - New resources are resources that exist in the target but not in the source
// - Updated resources are resources that exist in both but their data differ
// - Removed resources are resources that exist in the source but not in the target
// - Unmodified resources are resources that exist in both with identical data
func ComputeDiff(source *resources.Graph, target *resources.Graph) *Diff {
	newResources := []string{}
	removedResources := []string{}
	updatedResources := map[string]ResourceDiff{}
	unmodifiedResources := []string{}

	// Iterate over target resources to find new and updated resources
	for urn, r := range target.Resources() {
		if sourceResource, exists := source.GetResource(urn); !exists {
			// Resource is new if it doesn't exist in the source
			newResources = append(newResources, urn)
		} else {
			// Check if resource is updated or unmodified
			propertyDiffs := CompareData(sourceResource.Data(), r.Data())
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
