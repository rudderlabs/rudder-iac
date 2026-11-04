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
	FlowJSONMapper     Flow = "json_mapper"
	FlowObjectMapping  Flow = "object_mapping"
	FlowVisualMapperV2 Flow = "visual_mapper_v2"
)

// UsesDestinationSpecificFlow reports whether a destination drives rETL
// through its own flow rather than the generic ones. The list mirrors the
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
//
// The visual-mapper-v2 flow is read off the destination's own
// supportsVisualMapperV2 metadata rather than a list of destination names, so
// a destination that gains the flag upstream needs no change here. It takes
// precedence because the two flags are exclusive upstream, and it carries no
// object: a vdm-next destination declares its field set through its own
// config, which is why it is the one flow a sync can mirror on.
func ClassifyFlow(apiType string, supportsVisualMapper, supportsVisualMapperV2 bool, object *string) (Flow, error) {
	if UsesDestinationSpecificFlow(apiType) {
		return "", fmt.Errorf("destination api type %q uses a destination-specific rETL flow, which is not supported", apiType)
	}
	if supportsVisualMapperV2 {
		if object != nil {
			return "", fmt.Errorf("'object' is not allowed: destination api type %q maps fields through its own destination config, not an object", apiType)
		}
		return FlowVisualMapperV2, nil
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
