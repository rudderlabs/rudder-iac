package common

import (
	"fmt"
	"maps"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Connection modes that route events through the destination's own SDK. A
// source type offering either is the one entitled to a native-SDK toggle.
const (
	ConnectionModeDevice = "device"
	ConnectionModeHybrid = "hybrid"
)

// NativeSDKSourceTypes returns the source types entitled to a use_native_sdk
// entry, derived from connectionModes. Upstream keeps the same invariant —
// schema.json's useNativeSDK keys are the device-capable subset of
// supportedConnectionModes — so the block needs no source-type list of its own.
func NativeSDKSourceTypes(connectionModes map[string][]string) []string {
	var sourceTypes []string
	for _, localSourceType := range slices.Sorted(maps.Keys(connectionModes)) {
		modes := connectionModes[localSourceType]
		if slices.Contains(modes, ConnectionModeDevice) || slices.Contains(modes, ConnectionModeHybrid) {
			sourceTypes = append(sourceTypes, localSourceType)
		}
	}
	return sourceTypes
}

// NativeSDKProperties returns ConfigProperty entries for the use_native_sdk
// block. As in ConnectionModeProperties the API spelling comes from
// apiSourceType, so definitions never hand-write camelCase source types.
func NativeSDKProperties(connectionModes map[string][]string) []converter.ConfigProperty {
	sourceTypes := NativeSDKSourceTypes(connectionModes)
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
			fmt.Sprintf("useNativeSDK.%s", remoteSourceType),
			fmt.Sprintf("use_native_sdk.%s", localSourceType),
		))
	}

	return properties
}
