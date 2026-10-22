package common

import (
	"fmt"
	"regexp"
	"slices"
)

const (
	// ConsentProviderCustom is the provider whose entries need a
	// resolution_strategy.
	ConsentProviderCustom = "custom"
	// ConsentValueRegex is what each value in an entry's consents must match:
	// a `{{ path || fallback }}` template, an `env.` value, or at most 100
	// characters.
	ConsentValueRegex = `^(\{\{.*\|\|.*\}\}|env\..+|.{0,100})$`
	// ConsentValueMessage is the validation message for a consent value that
	// does not match ConsentValueRegex.
	ConsentValueMessage = "must be at most 100 characters"
)

var (
	consentProviders            = []string{ConsentProviderCustom, "iubenda", "ketch", "oneTrust"}
	consentResolutionStrategies = []string{"and", "or"}
	consentValuePattern         = regexp.MustCompile(ConsentValueRegex)
)

// ConsentProviders returns a copy of the provider values ValidateConsentEntries
// accepts, for Describe to document.
func ConsentProviders() []string {
	return slices.Clone(consentProviders)
}

// ConsentResolutionStrategies returns the values resolution_strategy accepts
// when the provider is custom. Other providers' entries leave it unchecked.
func ConsentResolutionStrategies() []string {
	return slices.Clone(consentResolutionStrategies)
}

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
			Message: fmt.Sprintf("'provider' must be one of %v", consentProviders),
		}}
	}
	if entry.Provider != ConsentProviderCustom {
		return nil
	}
	if entry.ResolutionStrategy == "" {
		return []ConsentValidationError{{
			Path:    fmt.Sprintf("/%d/resolution_strategy", index),
			Message: "'resolution_strategy' is required when 'provider' is custom",
		}}
	}
	if !slices.Contains(consentResolutionStrategies, entry.ResolutionStrategy) {
		return []ConsentValidationError{{
			Path:    fmt.Sprintf("/%d/resolution_strategy", index),
			Message: fmt.Sprintf("'resolution_strategy' must be one of %v", consentResolutionStrategies),
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
			Message: "'consent' " + ConsentValueMessage,
		})
	}
	return errors
}

func isValidConsentProvider(provider string) bool {
	return slices.Contains(consentProviders, provider)
}
