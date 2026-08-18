package common

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	consentManagementLocalKey        = "consent_management"
	oneTrustCookieCategoriesLocalKey = "one_trust_cookie_categories"
	ketchConsentPurposesLocalKey     = "ketch_consent_purposes"
)

// Properties returns ConfigProperty entries for the consent_management block scoped to source types.
func Properties(sourceTypes []string) []converter.ConfigProperty {
	return sourceTypeArrayProperties(
		sourceTypes,
		"consentManagement",
		consentManagementLocalKey,
		map[string]any{
			"provider":           "provider",
			"resolutionStrategy": "resolution_strategy",
			"consents": converter.APINestedObject{
				LocalKey:  "consents",
				NestedKey: "consent",
			},
		},
	)
}

// OneTrustCookieCategoryProperties returns mappings for the schema-backed oneTrustCookieCategories block.
func OneTrustCookieCategoryProperties(sourceTypes []string) []converter.ConfigProperty {
	return sourceTypeArrayProperties(
		sourceTypes,
		"oneTrustCookieCategories",
		oneTrustCookieCategoriesLocalKey,
		map[string]any{"oneTrustCookieCategory": "one_trust_cookie_category"},
	)
}

// KetchConsentPurposeProperties returns mappings for the schema-backed ketchConsentPurposes block.
func KetchConsentPurposeProperties(sourceTypes []string) []converter.ConfigProperty {
	return sourceTypeArrayProperties(
		sourceTypes,
		"ketchConsentPurposes",
		ketchConsentPurposesLocalKey,
		map[string]any{"purpose": "purpose"},
	)
}

func sourceTypeArrayProperties(
	sourceTypes []string,
	apiRootKey string,
	localRootKey string,
	fields map[string]any,
) []converter.ConfigProperty {
	if len(sourceTypes) == 0 {
		return nil
	}

	properties := make([]converter.ConfigProperty, 0, len(sourceTypes))
	for _, localSourceType := range sourceTypes {
		remoteSourceType, ok := apiSourceType(localSourceType)
		if !ok {
			continue
		}
		properties = append(properties, converter.ArrayWithObjects(
			fmt.Sprintf("%s.%s", apiRootKey, remoteSourceType),
			fmt.Sprintf("%s.%s", localRootKey, localSourceType),
			fields,
		))
	}

	return properties
}
