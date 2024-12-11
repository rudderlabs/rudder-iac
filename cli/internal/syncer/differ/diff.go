package differ

import "github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"

// Diff represents the differences between two resource graphs
type Diff struct {
	// NewResources contains URNs of resources that exist in target but not in source
	NewResources []string
	// UpdatedResources contains URNs of resources that exist in both graphs but have different data
	UpdatedResources []string
	// RemovedResources contains URNs of resources that exist in source but not in target
	RemovedResources []string
	// UnmodifiedResources contains URNs of resources that exist in both graphs with identical data
	UnmodifiedResources []string
}

type ResourceDiff struct {
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
	updatedResources := []string{}
	unmodifiedResources := []string{}

	// Iterate over target resources to find new and updated resources
	for urn, r := range target.Resources() {
		if sourceResource, exists := source.GetResource(urn); !exists {
			// Resource is new if it doesn't exist in the source
			newResources = append(newResources, urn)
		} else {
			// Check if resource is updated or unmodified
			resourceDiff := CompareData(r.Data(), sourceResource.Data())
			if len(resourceDiff.Diffs) > 0 {
				updatedResources = append(updatedResources, urn)
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
func CompareData(r1, r2 resources.ResourceData) *ResourceDiff {
	diffs := make(map[string]PropertyDiff)

	// Iterate over properties in r1 to find differences
	for key, value1 := range r1 {
		if value2, exists := r2[key]; !exists || value1 != value2 {
			// Property is different if it doesn't exist in r2 or values are not equal
			diffs[key] = PropertyDiff{Property: key, SourceValue: value1, TargetValue: value2}
		}
	}

	// Iterate over properties in r2 to find properties that are not in r1
	for key, value2 := range r2 {
		if value1, exists := r1[key]; !exists {
			// Property is different if it doesn't exist in r1
			diffs[key] = PropertyDiff{Property: key, SourceValue: value1, TargetValue: value2}
		}
	}

	return &ResourceDiff{Diffs: diffs}
}
