package utils

import "fmt"

// CheckDuplicates detects duplicate values in a slice using a key extraction function
func CheckDuplicates[T any](items []T, keyFn func(T) string, errorPrefix string) error {
	seen := make(map[string]bool)

	for _, item := range items {
		key := keyFn(item)
		if key == "" {
			continue // Skip empty keys
		}

		if seen[key] {
			return fmt.Errorf("%s: duplicate %s", errorPrefix, key)
		}
		seen[key] = true
	}

	return nil
}

// CheckDuplicatesWithReference detects duplicates and returns detailed error information
func CheckDuplicatesWithReference[T any](items []T, keyFn func(T) string, referenceFn func(T) string, errorPrefix string) []DuplicateError {
	seen := make(map[string]T)
	var errors []DuplicateError

	for _, item := range items {
		key := keyFn(item)
		if key == "" {
			continue // Skip empty keys
		}

		if existing, exists := seen[key]; exists {
			errors = append(errors, DuplicateError{
				Key:          key,
				FirstRef:     referenceFn(existing),
				DuplicateRef: referenceFn(item),
				ErrorMessage: fmt.Sprintf("%s: duplicate %s", errorPrefix, key),
			})
		} else {
			seen[key] = item
		}
	}

	return errors
}

// DuplicateError represents a duplicate detection error with references
type DuplicateError struct {
	Key          string
	FirstRef     string
	DuplicateRef string
	ErrorMessage string
}

// Error implements the error interface
func (e DuplicateError) Error() string {
	return e.ErrorMessage
}

// HasDuplicates quickly checks if a slice has duplicates without detailed error reporting
func HasDuplicates[T comparable](items []T) bool {
	seen := make(map[T]bool)

	for _, item := range items {
		if seen[item] {
			return true
		}
		seen[item] = true
	}

	return false
}

// FindDuplicates returns all duplicate values in a slice
func FindDuplicates[T comparable](items []T) []T {
	seen := make(map[T]bool)
	duplicates := make(map[T]bool)
	var result []T

	for _, item := range items {
		if seen[item] {
			if !duplicates[item] {
				result = append(result, item)
				duplicates[item] = true
			}
		} else {
			seen[item] = true
		}
	}

	return result
}
