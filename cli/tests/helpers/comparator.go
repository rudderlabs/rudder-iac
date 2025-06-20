package helpers

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/samber/lo"
)

type Errors []error

func (e Errors) Error() string {
	return strings.Join(lo.Map(e, func(item error, _ int) string {
		return item.Error()
	}), "\n")
}

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

func compareSlices(path string, actual, expected reflect.Value, ignore []string, errors []error) []error {
	if actual.Len() != expected.Len() {
		errors = append(errors, fmt.Errorf("mismatch at path '%s': slice length differs, got %d, want %d", path, actual.Len(), expected.Len()))
	}

	minLen := min(expected.Len(), actual.Len())

	for i := range minLen {
		currentPath := fmt.Sprintf("%s[%d]", path, i)

		errors = compareValues(
			currentPath,
			actual.Index(i),
			expected.Index(i),
			ignore,
			errors,
		)
	}

	return errors
}

func buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return fmt.Sprintf("%s.%s", basePath, key)
}
