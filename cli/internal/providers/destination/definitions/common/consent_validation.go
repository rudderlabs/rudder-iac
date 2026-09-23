package common

import (
	"fmt"
	"regexp"
)

var consentValuePattern = regexp.MustCompile(`^(\{\{.*\|\|.*\}\}|env\..+|.{0,100})$`)

// ConsentManagement stores consent entries by local source type.
type ConsentManagement map[string][]ConsentEntry

// ConsentEntry is one consent-provider configuration.
type ConsentEntry struct {
	Provider           string   `mapstructure:"provider"`
	ResolutionStrategy string   `mapstructure:"resolution_strategy"`
	Consents           []string `mapstructure:"consents"`
}

// ConsentValidationError reports a validation error relative to one source block.
type ConsentValidationError struct {
	Path    string
	Message string
}

// ConsentValidator validates all consent entries for one source type.
type ConsentValidator func([]ConsentEntry) []ConsentValidationError

// ValidateConsentEntries applies canonical consent-management validation.
func ValidateConsentEntries(entries []ConsentEntry) []ConsentValidationError {
	var (
		errors        []ConsentValidationError
		seenProviders = make(map[string]struct{}, len(entries))
	)
	for index, entry := range entries {
		errors = append(errors, validateConsentEntry(index, entry)...)
		errors = append(errors, validateConsentValues(index, entry.Consents)...)
		if !isValidConsentProvider(entry.Provider) {
			continue
		}
		if _, exists := seenProviders[entry.Provider]; exists {
			errors = append(errors, ConsentValidationError{
				Path:    fmt.Sprintf("/%d/provider", index),
				Message: "only one consent entry can be configured per provider",
			})
			continue
		}
		seenProviders[entry.Provider] = struct{}{}
	}
	return errors
}

func validateConsentEntry(index int, entry ConsentEntry) []ConsentValidationError {
	if !isValidConsentProvider(entry.Provider) {
		return []ConsentValidationError{{
			Path:    fmt.Sprintf("/%d/provider", index),
			Message: "'provider' must be one of [custom iubenda ketch oneTrust]",
		}}
	}
	if entry.Provider != "custom" {
		return nil
	}
	if entry.ResolutionStrategy == "" {
		return []ConsentValidationError{{
			Path:    fmt.Sprintf("/%d/resolution_strategy", index),
			Message: "'resolution_strategy' is required when 'provider' is custom",
		}}
	}
	if entry.ResolutionStrategy != "and" && entry.ResolutionStrategy != "or" {
		return []ConsentValidationError{{
			Path:    fmt.Sprintf("/%d/resolution_strategy", index),
			Message: "'resolution_strategy' must be one of [and or]",
		}}
	}
	return nil
}

func validateConsentValues(entryIndex int, consents []string) []ConsentValidationError {
	var errors []ConsentValidationError
	for consentIndex, consent := range consents {
		if consentValuePattern.MatchString(consent) {
			continue
		}
		errors = append(errors, ConsentValidationError{
			Path:    fmt.Sprintf("/%d/consents/%d", entryIndex, consentIndex),
			Message: "'consent' must be at most 100 characters",
		})
	}
	return errors
}

func isValidConsentProvider(provider string) bool {
	switch provider {
	case "custom", "iubenda", "ketch", "oneTrust":
		return true
	default:
		return false
	}
}
