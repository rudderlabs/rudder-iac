package common

import (
	"fmt"
	"strings"
)

// Source definition categories that affect source-type resolution, mirroring
// the backend's SOURCE_CATEGORIES (rudder-config-backend
// src/utils/constants.ts). Only rETL-side source definitions (cloud-extract,
// singer, warehouse-action) carry these categories; the SDK source
// definitions the CLI event-stream provider manages (javascript, android,
// ...) have a null category upstream, so callers pass an empty category for
// them and resolution falls through to the type switch.
const (
	SourceCategoryCloud     = "cloud"
	SourceCategorySinger    = "singer-protocol"
	SourceCategoryWarehouse = "warehouse"
)

const (
	SourceTypeAMP           = "amp"
	SourceTypeAndroid       = "android"
	SourceTypeAndroidKotlin = "android_kotlin"
	SourceTypeCloud         = "cloud"
	SourceTypeCloudSource   = "cloud_source"
	SourceTypeCordova       = "cordova"
	SourceTypeFlutter       = "flutter"
	SourceTypeIOS           = "ios"
	SourceTypeIOSSwift      = "ios_swift"
	SourceTypeReactNative   = "react_native"
	SourceTypeShopify       = "shopify"
	SourceTypeUnity         = "unity"
	SourceTypeWarehouse     = "warehouse"
	SourceTypeWeb           = "web"
)

var apiSourceTypeByLocal = map[string]string{
	SourceTypeAMP:           "amp",
	SourceTypeAndroid:       "android",
	SourceTypeAndroidKotlin: "androidKotlin",
	SourceTypeCloud:         "cloud",
	SourceTypeCloudSource:   "cloudSource",
	SourceTypeCordova:       "cordova",
	SourceTypeFlutter:       "flutter",
	SourceTypeIOS:           "ios",
	SourceTypeIOSSwift:      "iosSwift",
	SourceTypeReactNative:   "reactnative",
	SourceTypeShopify:       "shopify",
	SourceTypeUnity:         "unity",
	SourceTypeWarehouse:     "warehouse",
	SourceTypeWeb:           "web",
}

func apiSourceType(typ string) (string, bool) {
	apiSourceType, ok := apiSourceTypeByLocal[typ]
	return apiSourceType, ok
}

// LocalToAPISourceTypes returns a copy of the canonical local→API source-type map.
func LocalToAPISourceTypes() map[string]string {
	out := make(map[string]string, len(apiSourceTypeByLocal))
	for local, api := range apiSourceTypeByLocal {
		out[local] = api
	}
	return out
}

// SourceTypeToken maps a source's definition type and category to the local
// source type a destination declares support for. It mirrors the backend's
// ServiceUtil.getSourceType (rudder-config-backend src/services/serviceUtil.ts),
// which produces the token used for the supportedSourceTypes membership check
// and as the key into supportedSourcesValidation at connect time — translated
// here to the CLI's local (snake_case) source types. Some SDK source types have
// different spellings across APIs: config-backend uses android_kotlin,
// ios_swift, and reactnative, while event-stream uses kotlin, swift, and
// react_native. Accept both spellings so imported event-stream specs validate
// against the destination source-type keys they actually require.
func SourceTypeToken(sourceType, category string) string {
	switch category {
	case SourceCategoryCloud, SourceCategorySinger:
		return SourceTypeCloudSource
	case SourceCategoryWarehouse:
		return SourceTypeWarehouse
	}

	switch strings.ToLower(sourceType) {
	case "javascript":
		return SourceTypeWeb
	case "android_kotlin", "kotlin":
		return SourceTypeAndroidKotlin
	case "ios_swift", "swift":
		return SourceTypeIOSSwift
	case "android":
		return SourceTypeAndroid
	case "ios":
		return SourceTypeIOS
	case "unity":
		return SourceTypeUnity
	case "reactnative", "react_native":
		return SourceTypeReactNative
	case "amp":
		return SourceTypeAMP
	case "flutter":
		return SourceTypeFlutter
	case "cordova":
		return SourceTypeCordova
	case "shopify":
		return SourceTypeShopify
	default:
		// Webhook sources and every other event-stream type (java, go, ...)
		// connect as cloud sources.
		return SourceTypeCloud
	}
}

// ValidateSourceTypes verifies that every local source type has an API config key.
func ValidateSourceTypes(sourceTypes []string) error {
	for _, localSourceType := range sourceTypes {
		if _, ok := apiSourceType(localSourceType); !ok {
			return fmt.Errorf("local source type %q has no API mapping", localSourceType)
		}
	}
	return nil
}
