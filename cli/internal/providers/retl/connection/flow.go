package connection

import (
	"errors"
	"fmt"
	"slices"
)

// Flow is how a connection maps source rows onto the destination. The spec
// never names one: it follows from the destination and from whether the spec
// sets an object, exactly as the backend derives it.
//
// Classify once, then branch on the result: each flow allows a different set
// of config fields.
type Flow string

const (
	FlowJSONMapper    Flow = "json_mapper"
	FlowObjectMapping Flow = "object_mapping"
)

// UsesDestinationSpecificFlow reports whether a destination drives rETL
// through its own flow rather than the two generic ones. The list mirrors the
// config-backend DESTINATION_SPECIFIC_REGISTRY in
// src/modules/retl/api-gateway/connection-config/constants.ts.
func UsesDestinationSpecificFlow(apiType string) bool {
	return slices.Contains([]string{"CUSTOMERIO", "CUSTOMERIO_AUDIENCE"}, apiType)
}

// ClassifyFlow decides which flow a connection to the given destination runs,
// from the destination's API type and visual mapper support and the config's
// object. It errors instead of guessing whenever the combination has no flow:
// destination-specific flows are not supported at all, and an object only
// means object mapping on a destination that supports the visual mapper.
func ClassifyFlow(apiType string, supportsVisualMapper bool, object *string) (Flow, error) {
	if UsesDestinationSpecificFlow(apiType) {
		return "", fmt.Errorf("destination api type %q uses a destination-specific rETL flow, which is not supported", apiType)
	}
	if object == nil {
		return FlowJSONMapper, nil
	}
	if !supportsVisualMapper {
		return "", fmt.Errorf("'object' is not allowed: destination api type %q does not support object mapping", apiType)
	}
	if *object == "" {
		return "", errors.New("'object' must not be empty")
	}
	return FlowObjectMapping, nil
}
