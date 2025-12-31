package reporters

import (
	"fmt"
	"reflect"
	"sort"
)

// ValuePair holds a source/target value pair for diff comparison
type ValuePair struct {
	Source any
	Target any
}

// ComputeNestedDiffs recursively compares two values and returns a map of
// dot-notation paths to their value pairs for all leaf-level differences.
//
// Examples:
//   - Maps: {a:1, b:2} vs {a:1, b:3} → {"b": {2, 3}}
//   - Arrays: [1, 2] vs [2, 1] → {"0": {1, 2}, "1": {2, 1}}
//   - Nested: {x:{y:1}} vs {x:{y:2}} → {"x.y": {1, 2}}
//   - Mixed: {a:1, items:[1,2]} vs {a:1, items:[1,3]} → {"items.1": {2, 3}}
func ComputeNestedDiffs(source, target any) map[string]ValuePair {
	return flattenDiffs("", source, target)
}

// flattenDiffs is the recursive implementation that compares values and builds paths
func flattenDiffs(basePath string, source, target any) map[string]ValuePair {
	result := make(map[string]ValuePair)

	// Handle nil cases
	if source == nil && target == nil {
		return result // Both nil, no diff
	}
	if source == nil || target == nil {
		// One is nil, record the diff
		result[basePath] = ValuePair{Source: source, Target: target}
		return result
	}

	// Get reflection values to check types
	sourceVal := reflect.ValueOf(source)
	targetVal := reflect.ValueOf(target)

	// Handle type mismatches
	if sourceVal.Type() != targetVal.Type() {
		result[basePath] = ValuePair{Source: source, Target: target}
		return result
	}

	// Try to cast to map[string]any
	sourceMap, sourceIsMap := toMap(source)
	targetMap, targetIsMap := toMap(target)

	if sourceIsMap && targetIsMap {
		// Both are maps - recurse on all keys
		return flattenMapDiffs(basePath, sourceMap, targetMap)
	}

	// Try to cast to []any
	sourceSlice, sourceIsSlice := toSlice(source)
	targetSlice, targetIsSlice := toSlice(target)

	if sourceIsSlice && targetIsSlice {
		// Both are slices - recurse on all indices
		return flattenSliceDiffs(basePath, sourceSlice, targetSlice)
	}

	// Primitive comparison
	if !reflect.DeepEqual(source, target) {
		result[basePath] = ValuePair{Source: source, Target: target}
	}

	return result
}

// flattenMapDiffs recursively compares two maps
func flattenMapDiffs(basePath string, sourceMap, targetMap map[string]any) map[string]ValuePair {
	result := make(map[string]ValuePair)

	// Get union of all keys
	allKeys := make(map[string]bool)
	for k := range sourceMap {
		allKeys[k] = true
	}
	for k := range targetMap {
		allKeys[k] = true
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Recurse on each key
	for _, key := range keys {
		sourceVal, sourceHas := sourceMap[key]
		targetVal, targetHas := targetMap[key]

		var sourceItem, targetItem any
		if sourceHas {
			sourceItem = sourceVal
		} else {
			sourceItem = nil
		}
		if targetHas {
			targetItem = targetVal
		} else {
			targetItem = nil
		}

		// Build new path
		newPath := buildPath(basePath, key)

		// Recursively compare
		subDiffs := flattenDiffs(newPath, sourceItem, targetItem)
		for path, pair := range subDiffs {
			result[path] = pair
		}
	}

	return result
}

// flattenSliceDiffs recursively compares two slices
func flattenSliceDiffs(basePath string, sourceSlice, targetSlice []any) map[string]ValuePair {
	result := make(map[string]ValuePair)

	maxLen := len(sourceSlice)
	if len(targetSlice) > maxLen {
		maxLen = len(targetSlice)
	}

	// Compare each index
	for i := 0; i < maxLen; i++ {
		var sourceItem, targetItem any

		if i < len(sourceSlice) {
			sourceItem = sourceSlice[i]
		} else {
			sourceItem = nil
		}

		if i < len(targetSlice) {
			targetItem = targetSlice[i]
		} else {
			targetItem = nil
		}

		// Build new path with index
		newPath := buildPath(basePath, fmt.Sprintf("%d", i))

		// Recursively compare
		subDiffs := flattenDiffs(newPath, sourceItem, targetItem)
		for path, pair := range subDiffs {
			result[path] = pair
		}
	}

	return result
}

// buildPath constructs a dot-notation path
func buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return basePath + "." + key
}

// toMap attempts to convert a value to map[string]any
func toMap(val any) (map[string]any, bool) {
	if m, ok := val.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

// toSlice attempts to convert a value to []any
func toSlice(val any) ([]any, bool) {
	// Direct cast
	if s, ok := val.([]any); ok {
		return s, true
	}

	// Use reflection for other slice types
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Slice {
		return nil, false
	}

	// Convert to []any
	result := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}
	return result, true
}
