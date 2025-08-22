package helpers

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/samber/lo"
)

type Errors []error

func (e Errors) Error() string {
	return strings.Join(lo.Map(e, func(item error, _ int) string {
		return item.Error()
	}), "\n")
}

const (
	STABLE_KEY_NAME = "name"
)

// CompareStates recursively compares two interface{} values while ignoring specified fields
// It returns an error containing all differences found, nil if they match
func CompareStates(actual, expected any, ignore []string) error {
	var errors Errors

	errors = compareValues("", reflect.ValueOf(actual), reflect.ValueOf(expected), ignore, errors)
	if len(errors) == 0 {
		return nil
	}
	return errors
}

// compareValues recursively compares two reflect.Value objects and collects errors
func compareValues(path string, actual, expected reflect.Value, ignore []string, errors []error) []error {

	if slices.Contains(ignore, path) {
		return errors
	}

	if !actual.IsValid() && !expected.IsValid() {
		return errors
	}

	if !actual.IsValid() {
		errors = append(errors, fmt.Errorf("mismatch at path '%s': got <nil>, want %v", path, expected.Interface()))
		return errors
	}

	if !expected.IsValid() {
		errors = append(errors, fmt.Errorf("mismatch at path '%s': got %v, want <nil>", path, actual.Interface()))
		return errors
	}

	if actual.Type() != expected.Type() {
		errors = append(errors, fmt.Errorf("mismatch at path '%s': type mismatch, got %T, want %T", path, actual.Interface(), expected.Interface()))
		return errors
	}

	switch actual.Kind() {
	case reflect.Map:
		errors = compareMaps(path, actual, expected, ignore, errors)

	case reflect.Slice:
		errors = compareSlices(path, actual, expected, ignore, errors)

	case reflect.Interface:
		errors = compareValues(path, actual.Elem(), expected.Elem(), ignore, errors)

	default:
		if !reflect.DeepEqual(actual.Interface(), expected.Interface()) {
			errors = append(errors, fmt.Errorf("mismatch at path '%s': got %v, want %v", path, actual.Interface(), expected.Interface()))
		}
	}

	return errors
}

func compareMaps(path string, actual, expected reflect.Value, ignore []string, errors []error) []error {
	actualMap := actual.Interface().(map[string]any)
	expectedMap := expected.Interface().(map[string]any)

	if len(actualMap) != len(expectedMap) {
		errors = append(errors, fmt.Errorf("mismatch at path '%s': map key count differs, got %d keys, want %d keys", path, len(actualMap), len(expectedMap)))
	}

	for key, expectedValue := range expectedMap {
		currentPath := buildPath(path, key)

		actualValue, exists := actualMap[key]
		if !exists {
			errors = append(errors, fmt.Errorf("mismatch at path '%s': missing key '%s' in actual", path, key))
			continue
		}

		errors = compareValues(currentPath, reflect.ValueOf(actualValue), reflect.ValueOf(expectedValue), ignore, errors)
	}

	for key := range actualMap {
		_, exists := expectedMap[key]
		if !exists {
			errors = append(errors, fmt.Errorf("mismatch at path '%s': extra key '%s' in actual", path, key))
		}
	}

	return errors
}

// compareSlices compares two slices and returns an error if they differ
// It ignores the order of elements in the slices if the elements are orderable
// i.e. primitive types by sorting them first
func compareSlices(path string, actual, expected reflect.Value, ignore []string, errors []error) []error {
	if actual.Len() != expected.Len() {
		errors = append(errors, fmt.Errorf("mismatch at path '%s': slice length differs, got %d, want %d", path, actual.Len(), expected.Len()))
	}

	actualSorted := sortSlice(actual)
	expectedSorted := sortSlice(expected)

	minLen := min(expectedSorted.Len(), actualSorted.Len())

	for i := range minLen {
		currentPath := fmt.Sprintf("%s[%d]", path, i)

		errors = compareValues(
			currentPath,
			actualSorted.Index(i),
			expectedSorted.Index(i),
			ignore,
			errors,
		)
	}

	return errors
}

// sortSlice creates a sorted copy of the slice for consistent comparison
// It only sorts slices containing elements with ordered constraints (strings, numbers, bools)
// Complex types like maps and structs are left in their original order
func sortSlice(input reflect.Value) reflect.Value {
	orderable, kind := isSliceOrderable(input)
	if !orderable {
		return input
	}

	anyslice := make([]any, input.Len())
	for i := 0; i < input.Len(); i++ {
		anyslice[i] = input.Index(i).Interface()
	}

	if kind == reflect.Map {
		anyslice = sortMapSlice(anyslice, STABLE_KEY_NAME)
	} else {
		anyslice = sortPrimitiveSlice(anyslice)
	}

	sorted := reflect.MakeSlice(input.Type(), len(anyslice), len(anyslice))
	for i, val := range anyslice {
		sorted.Index(i).Set(reflect.ValueOf(val))
	}

	return sorted

}

func sortPrimitiveSlice(input []any) []any {
	// Sort based on string representation for consistent ordering
	// which is a predictable way to sort slices
	sort.Slice(input, func(i, j int) bool {
		return fmt.Sprintf("%v", input[i]) < fmt.Sprintf("%v", input[j])
	})
	return input
}

func sortMapSlice(input []any, stableKey string) []any {
	// Maps with the stable key: Grouped at the beginning, sorted alphabetically
	// Maps without the stable key: Grouped at the end, in original order
	sort.Slice(input, func(i, j int) bool {
		iVal, iOk := input[i].(map[string]any)[stableKey]
		jVal, jOk := input[j].(map[string]any)[stableKey]

		// If both have the key, sort by value
		if iOk && jOk {
			return fmt.Sprintf("%v", iVal) < fmt.Sprintf("%v", jVal)
		}

		// If only i has the key, i comes first
		if iOk && !jOk {
			return true
		}

		// If only j has the key, j comes first
		if !iOk && jOk {
			return false
		}

		// If neither has the key, maintain original order
		return false
	})
	return input
}

// isSliceOrderable determines if a slice contains elements that have ordered constraints
// Returns true for slices of primitive types (strings, numbers, bools)
// Returns false for slices of complex types (maps, structs, interfaces, etc.)
func isSliceOrderable(slice reflect.Value) (bool, reflect.Kind) {
	if slice.Len() == 0 {
		return false, reflect.Invalid // Empty slices are not orderable(there is nothing to sort)
	}

	firstElem := slice.Index(0)

	if firstElem.Kind() == reflect.Interface {
		firstElem = firstElem.Elem()
	}

	kind := firstElem.Kind()
	switch kind {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.Map:
		return true, kind
	}

	return false, kind
}

func buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return fmt.Sprintf("%s.%s", basePath, key)
}
