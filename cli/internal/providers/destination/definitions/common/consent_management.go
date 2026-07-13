package common

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const consentManagementLocalKey = "consent_management"

// Properties returns ConfigProperty entries for the consent_management block scoped to source types.
func Properties(sourceTypes []string) []converter.ConfigProperty {
	if len(sourceTypes) == 0 {
		return nil
	}

	properties := make([]converter.ConfigProperty, 0, len(sourceTypes))
	for _, sourceType := range sourceTypes {
		remoteSourceType, ok := apiSourceType(sourceType)
		if !ok {
			continue
		}
		properties = append(properties, converter.ArrayWithObjects(
			fmt.Sprintf("consentManagement.%s", remoteSourceType),
			fmt.Sprintf("%s.%s", consentManagementLocalKey, sourceType),
			map[string]any{
				"provider":           "provider",
				"resolutionStrategy": "resolution_strategy",
				"consents": converter.APINestedObject{
					LocalKey:  "consents",
					NestedKey: "consent",
				},
			},
		))
	}

	return properties
}
