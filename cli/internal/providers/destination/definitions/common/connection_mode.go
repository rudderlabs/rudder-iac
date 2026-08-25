package common

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// ConnectionMode stores the connection mode value by local source type, e.g.
// {"web": "device"}. The property mapping copies values verbatim; the valid
// values differ per source type per destination and are enforced by
// validateConnectionMode against RegisteredDefinition.ConnectionModes.
type ConnectionMode map[string]string

// ConnectionModeProperties returns ConfigProperty entries for the
// connection_mode block scoped to source types. Unlike consent_management,
// the API shape is a flat string per source type, so each entry is a bare
// converter.Simple rather than a reshape.
func ConnectionModeProperties(sourceTypes []string) []converter.ConfigProperty {
	if len(sourceTypes) == 0 {
		return nil
	}

	properties := make([]converter.ConfigProperty, 0, len(sourceTypes))
	for _, localSourceType := range sourceTypes {
		remoteSourceType, ok := apiSourceType(localSourceType)
		if !ok {
			continue
		}
		properties = append(properties, converter.Simple(
			fmt.Sprintf("connectionMode.%s", remoteSourceType),
			fmt.Sprintf("connection_mode.%s", localSourceType),
		))
	}

	return properties
}
