package unflatten

import (
	"strconv"
	"strings"
)

// UnflattenSchema converts a flattened schema map to a nested structure
func UnflattenSchema(flattened map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range flattened {
		setNestedValue(result, strings.Split(key, "."), value)
	}

	return result
}

// setNestedValue recursively sets a value in a nested structure
func setNestedValue(obj map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	if len(path) == 1 {
		// Last element in path - set the value
		obj[path[0]] = value
		return
	}

	currentKey := path[0]
	remainingPath := path[1:]
	nextKey := remainingPath[0]

	// Check if next key is a numeric index (array)
	if isNumericIndex(nextKey) {
		// Current key should point to an array
		if obj[currentKey] == nil {
			obj[currentKey] = make([]interface{}, 0)
		}

		// Ensure it's an array
		arr, ok := obj[currentKey].([]interface{})
		if !ok {
			arr = make([]interface{}, 0)
			obj[currentKey] = arr
		}

		// Set value in array
		setArrayValue(&arr, remainingPath, value)
		obj[currentKey] = arr
	} else {
		// Current key should point to an object
		if obj[currentKey] == nil {
			obj[currentKey] = make(map[string]interface{})
		}

		// Ensure it's a map
		nestedObj, ok := obj[currentKey].(map[string]interface{})
		if !ok {
			nestedObj = make(map[string]interface{})
			obj[currentKey] = nestedObj
		}

		// Recursively set the value
		setNestedValue(nestedObj, remainingPath, value)
	}
}

// setArrayValue sets a value in an array at the specified index path
func setArrayValue(arr *[]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	indexStr := path[0]
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return // Invalid index
	}

	// Expand array if necessary
	for len(*arr) <= index {
		*arr = append(*arr, nil)
	}

	if len(path) == 1 {
		// Last element - set the value
		(*arr)[index] = value
		return
	}

	// More path elements remain
	remainingPath := path[1:]
	nextKey := remainingPath[0]

	if isNumericIndex(nextKey) {
		// Next level is also an array
		if (*arr)[index] == nil {
			(*arr)[index] = make([]interface{}, 0)
		}

		nestedArr, ok := (*arr)[index].([]interface{})
		if !ok {
			nestedArr = make([]interface{}, 0)
			(*arr)[index] = nestedArr
		}

		setArrayValue(&nestedArr, remainingPath, value)
		(*arr)[index] = nestedArr
	} else {
		// Next level is an object
		if (*arr)[index] == nil {
			(*arr)[index] = make(map[string]interface{})
		}

		nestedObj, ok := (*arr)[index].(map[string]interface{})
		if !ok {
			nestedObj = make(map[string]interface{})
			(*arr)[index] = nestedObj
		}

		setNestedValue(nestedObj, remainingPath, value)
	}
}

// isNumericIndex checks if a string represents a numeric array index
func isNumericIndex(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
