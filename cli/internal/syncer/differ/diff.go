package differ

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
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

// HasNonSecretDiff reports whether the plan contains any change a user actually made,
// ignoring resources that update only because they carry an unknown secret (which
// re-applies every run by design). Used by the import "project not synced" guard so
// the presence of secrets does not permanently block imports.
func (d *Diff) HasNonSecretDiff() bool {
	if len(d.NewResources) > 0 ||
		len(d.ImportableResources) > 0 ||
		len(d.RemovedResources) > 0 {
		return true
	}
	for _, rd := range d.UpdatedResources {
		if !rd.IsSecretOnly() {
			return true
		}
	}
	return false
}

type ResourceDiff struct {
	URN   string
	Diffs map[string]PropertyDiff
	// SecretOnly is true when every property diff is secret-driven, so the
	// resource is "always re-applied" rather than genuine drift. It is computed
	// once while the diffs are built (see compareData) and cached here.
	SecretOnly bool
}

// IsSecretOnly reports whether this resource updates only because of unknown
// secrets. The verdict is precomputed in ComputeDiff, so this is O(1).
func (rd ResourceDiff) IsSecretOnly() bool {
	return rd.SecretOnly
}

type PropertyDiff struct {
	Property    string
	SourceValue any
	TargetValue any
	// SecretOnly is true when this diff exists only because of an unknown secret, so
	// the reporter can render it distinctly and classify the resource as always
	// re-applied. It propagates up: a containing map diff is SecretOnly only when every
	// child diff is.
	SecretOnly bool
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
			var sData map[string]any
			var tData map[string]any

			if sourceResource.RawData() != nil && r.RawData() != nil {
				_ = mapstructure.Decode(sourceResource.RawData(), &sData)
				_ = mapstructure.Decode(r.RawData(), &tData)

			} else {
				sData = sourceResource.Data()
				tData = r.Data()
			}

			// Check if resource is updated or unmodified. secretOnly is computed
			// while the diffs are built and cached on the ResourceDiff.
			propertyDiffs, secretOnly := CompareData(sData, tData)
			if len(propertyDiffs) > 0 {
				updatedResources[urn] = ResourceDiff{URN: urn, Diffs: propertyDiffs, SecretOnly: secretOnly}
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

// CompareData compares the data of two resources and returns the differences and
// whether the whole set is secret-only.
// The secret-only verdict is computed as the diffs are built rather than by
// re-looping afterwards; callers propagate it up a nested map and cache it on a
// ResourceDiff (see ComputeDiff) so IsSecretOnly stays O(1).
func CompareData(r1, r2 resources.ResourceData) (map[string]PropertyDiff, bool) {
	diffs := make(map[string]PropertyDiff)

	// record is the single, once-per-key write path into diffs, so the running
	// count of secret diffs stays in lockstep with the map size.
	secretDiffs := 0
	record := func(key string, d PropertyDiff) {
		diffs[key] = d
		if d.SecretOnly {
			secretDiffs++
		}
	}

	// Helper function to compare values recursively
	var compareValues func(key string, v1, v2 any)
	compareValues = func(key string, v1, v2 any) {

		if isNil(v1) && isNil(v2) {
			return
		}

		// Rewrite both sides so a []any-of-objects from one decode path (e.g.
		// JSON remote state) and its equal counterpart from another (e.g. YAML
		// spec) land on the same type before the type-equality gate below.
		if newV1, ok := rewriteCompatibleType(v1); ok {
			v1 = newV1
		}

		if newV2, ok := rewriteCompatibleType(v2); ok {
			v2 = newV2
		}

		if reflect.TypeOf(v1) != reflect.TypeOf(v2) {
			record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2})
			return
		}

		// If v1 and v2 are pointers, compare the dereferenced values
		if reflect.TypeOf(v1).Kind() == reflect.Pointer {
			if isNil(v1) || isNil(v2) {
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2})
				return
			}
			compareValues(key, reflect.ValueOf(v1).Elem().Interface(), reflect.ValueOf(v2).Elem().Interface())
			return
		}

		switch v1Typed := v1.(type) {

		case *resources.PropertyRef:
			v2Typed := v2.(*resources.PropertyRef)
			if !comparePropertyRefs(v1Typed, v2Typed) {
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2})
			}
		case resources.PropertyRef:
			v2Typed := v2.(resources.PropertyRef)
			if !comparePropertyRefs(&v1Typed, &v2Typed) {
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2})
			}

		// A secret owns the "how do we compare?" decision in its own type: an
		// unknown secret always diffs (we can't read the remote, so we re-apply),
		// otherwise known values are compared. The diff is flagged Secret so the
		// reporter renders it distinctly and the resource is classified as always
		// re-applied. SourceValue/TargetValue stay secret.String so they mask. A
		// *secret.String (the form that survives RawData's struct→map decode, like
		// *PropertyRef) is dereferenced to this case by the pointer branch above.
		case secret.String:
			v2Typed := v2.(secret.String)
			if v1Typed.Diff(v2Typed) {
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2, SecretOnly: true})
			}

		// TODO: a secret.String nested inside a slice (same for []any below) is
		// compared whole via reflect.DeepEqual, so it bypasses the secret case
		// above — it still re-applies every run but isn't flagged SecretOnly, so it
		// renders as a normal masked update rather than under "always re-applied".
		// No value leaks (Format masks). Extend secret-awareness to slices.
		case []map[string]any:
			v2Typed := v2.([]map[string]any)
			if !reflect.DeepEqual(v1Typed, v2Typed) {
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2})
			}

		case map[string]any:
			v2Typed := v2.(map[string]any)
			subDiffs, subSecretOnly := CompareData(v1Typed, v2Typed)
			if len(subDiffs) > 0 {
				// A nested map is secret-only (always re-applied) only when every
				// child diff is itself secret-driven; one real sibling change makes
				// the whole map a real diff.
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2, SecretOnly: subSecretOnly})
			}
		case []any:
			v2Typed := v2.([]any)
			if !reflect.DeepEqual(v1Typed, v2Typed) {
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2})
			}
		default:
			if v1 != v2 {
				record(key, PropertyDiff{Property: key, SourceValue: v1, TargetValue: v2})
			}
		}
	}

	// Iterate over properties in r1 to find differences
	for key, value1 := range r1 {
		if value2, exists := r2[key]; !exists {
			// Presence-based secret wrapping omits keys the API strips. A secret
			// present on only one side is still secret-driven drift (re-apply),
			// not a real config change.
			record(key, PropertyDiff{Property: key, SourceValue: value1, TargetValue: nil, SecretOnly: isSecretValue(value1)})
		} else {
			compareValues(key, value1, value2)
		}
	}

	// Iterate over properties in r2 to find properties that are not in r1
	for key, value2 := range r2 {
		if _, exists := r1[key]; !exists {
			record(key, PropertyDiff{Property: key, SourceValue: nil, TargetValue: value2, SecretOnly: isSecretValue(value2)})
		}
	}

	// secretDiffs == len(diffs) means every recorded diff is secret-driven; the
	// > 0 guard keeps an empty diff set from counting as secret-only.
	return diffs, secretDiffs > 0 && secretDiffs == len(diffs)
}

// rewrite []any ->  map[string]any if possible
// and return back the response.
func rewriteCompatibleType(input any) (any, bool) {

	if _, ok := input.([]any); !ok {
		return nil, false
	}

	slice := input.([]any)

	output := make([]map[string]any, len(slice))
	for i, item := range slice {
		if _, ok := item.(map[string]any); !ok {
			return nil, false
		}
		output[i] = item.(map[string]any)
	}

	return output, true
}

func isNil(val any) bool {
	if val == nil {
		return true
	}

	if lo.Contains(isNilTypes, reflect.ValueOf(val).Kind()) {
		return reflect.ValueOf(val).IsNil()
	}

	return false
}

// isSecretValue reports whether v is a secret.String (value or pointer). Used
// when one side omits a key so a presence-based secret still classifies as
// SecretOnly rather than genuine drift.
func isSecretValue(v any) bool {
	switch v.(type) {
	case secret.String, *secret.String:
		return true
	default:
		return false
	}
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
