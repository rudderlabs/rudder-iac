package common

import "fmt"

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

// ValidateSourceTypes verifies that every local source type has an API config key.
func ValidateSourceTypes(sourceTypes []string) error {
	for _, sourceType := range sourceTypes {
		if _, ok := apiSourceType(sourceType); !ok {
			return fmt.Errorf("source type %q has no API mapping", sourceType)
		}
	}
	return nil
}
