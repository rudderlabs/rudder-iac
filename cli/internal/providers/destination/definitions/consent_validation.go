package definitions

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

func (d *RegisteredDefinition) validateConsentManagement(config map[string]any) []ConfigError {
	if _, ok := structFieldsByMapstructureTag(d.configType)["consent_management"]; !ok {
		return nil
	}

	consentManagement, errors := consentManagementBlock(config)
	if consentManagement == nil {
		return errors
	}

	for _, sourceType := range slices.Sorted(maps.Keys(consentManagement)) {
		if !slices.Contains(d.SourceTypes, sourceType) {
			errors = append(errors, unsupportedConsentSourceError(d, sourceType))
			continue
		}
		if consentManagement[sourceType] == nil {
			errors = append(errors, ConfigError{
				Path:    joinConfigPath("/consent_management", sourceType),
				Message: fmt.Sprintf("'%s' consent entries must be an array", sourceType),
			})
			continue
		}
		errors = append(
			errors,
			validateConsentEntriesShape(
				joinConfigPath("/consent_management", sourceType),
				consentManagement[sourceType],
			)...,
		)
		validator := common.ConsentValidator(common.ValidateConsentEntries)
		if override, ok := d.ConsentValidationOverrides[sourceType]; ok {
			validator = override
		}
		errors = append(errors, validateConsentEntries(sourceType, consentManagement[sourceType], validator)...)
	}
	return errors
}

func consentManagementBlock(config map[string]any) (map[string]any, []ConfigError) {
	rawConsentManagement, exists := config["consent_management"]
	if !exists {
		return nil, nil
	}
	if rawConsentManagement == nil {
		return nil, []ConfigError{{
			Path:    "/consent_management",
			Message: "'consent_management' must be an object",
		}}
	}

	consentManagement, _ := rawConsentManagement.(map[string]any)
	return consentManagement, nil
}

func validateConsentEntriesShape(basePath string, raw any) []ConfigError {
	entries, ok := raw.([]any)
	if !ok {
		return nil
	}

	var errors []ConfigError
	for index, rawEntry := range entries {
		entryPath := joinConfigPath(basePath, fmt.Sprintf("%d", index))
		errors = append(errors, validateConsentEntryShape(entryPath, rawEntry)...)
	}
	return errors
}

func validateConsentEntryShape(basePath string, raw any) []ConfigError {
	if raw == nil {
		return []ConfigError{{Path: basePath, Message: "consent entry must be an object"}}
	}
	entry, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	var errors []ConfigError
	if provider, exists := entry["provider"]; exists && provider == nil {
		errors = append(errors, ConfigError{
			Path:    joinConfigPath(basePath, "provider"),
			Message: "'provider' must be a string",
		})
	}
	errors = append(errors, validateConsentValuesShape(basePath, entry)...)
	return errors
}

func validateConsentValuesShape(basePath string, entry map[string]any) []ConfigError {
	rawConsents, exists := entry["consents"]
	if !exists {
		return nil
	}
	if rawConsents == nil {
		return []ConfigError{{
			Path:    joinConfigPath(basePath, "consents"),
			Message: "'consents' must be an array",
		}}
	}

	consents, ok := rawConsents.([]any)
	if !ok {
		return nil
	}
	var errors []ConfigError
	for index, consent := range consents {
		if consent == nil {
			errors = append(errors, ConfigError{
				Path:    joinConfigPath(joinConfigPath(basePath, "consents"), fmt.Sprintf("%d", index)),
				Message: "'consent' must be a string",
			})
		}
	}
	return errors
}

func unsupportedConsentSourceError(d *RegisteredDefinition, sourceType string) ConfigError {
	return ConfigError{
		Path: joinConfigPath("/consent_management", sourceType),
		Message: fmt.Sprintf(
			"source type '%s' is not supported by destination type '%s'; supported source types: %s",
			sourceType,
			d.Type,
			strings.Join(d.SourceTypes, ", "),
		),
	}
}

func validateConsentEntries(sourceType string, raw any, validator common.ConsentValidator) []ConfigError {
	var entries []common.ConsentEntry
	if err := decodeConfig(raw, &entries); err != nil {
		return nil
	}

	basePath := joinConfigPath("/consent_management", sourceType)
	validationErrors := validator(entries)
	errors := make([]ConfigError, 0, len(validationErrors))
	for _, validationError := range validationErrors {
		errors = append(errors, ConfigError{
			Path:    basePath + validationError.Path,
			Message: validationError.Message,
		})
	}
	return errors
}
