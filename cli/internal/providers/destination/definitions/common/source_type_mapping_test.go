package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

func TestSourceTypeConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{
		"amp",
		"android",
		"android_kotlin",
		"cloud",
		"cloud_source",
		"cordova",
		"flutter",
		"ios",
		"ios_swift",
		"react_native",
		"shopify",
		"unity",
		"warehouse",
		"web",
	}, []string{
		common.SourceTypeAMP,
		common.SourceTypeAndroid,
		common.SourceTypeAndroidKotlin,
		common.SourceTypeCloud,
		common.SourceTypeCloudSource,
		common.SourceTypeCordova,
		common.SourceTypeFlutter,
		common.SourceTypeIOS,
		common.SourceTypeIOSSwift,
		common.SourceTypeReactNative,
		common.SourceTypeShopify,
		common.SourceTypeUnity,
		common.SourceTypeWarehouse,
		common.SourceTypeWeb,
	})
}

func TestSourceTypeToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		sourceType string
		category   string
		want       string
	}{
		{name: "cloud category wins over type", sourceType: "javascript", category: common.SourceCategoryCloud, want: common.SourceTypeCloudSource},
		{name: "singer category", sourceType: "salesforce", category: common.SourceCategorySinger, want: common.SourceTypeCloudSource},
		{name: "warehouse category", sourceType: "snowflake", category: common.SourceCategoryWarehouse, want: common.SourceTypeWarehouse},
		{name: "javascript", sourceType: "javascript", want: common.SourceTypeWeb},
		{name: "javascript case-insensitive", sourceType: "Javascript", want: common.SourceTypeWeb},
		{name: "android_kotlin", sourceType: "android_kotlin", want: common.SourceTypeAndroidKotlin},
		{name: "ios_swift", sourceType: "ios_swift", want: common.SourceTypeIOSSwift},
		{name: "android", sourceType: "android", want: common.SourceTypeAndroid},
		{name: "ios", sourceType: "iOS", want: common.SourceTypeIOS},
		{name: "unity", sourceType: "unity", want: common.SourceTypeUnity},
		{name: "backend reactnative spelling", sourceType: "ReactNative", want: common.SourceTypeReactNative},
		{name: "cli react_native spelling", sourceType: "react_native", want: common.SourceTypeReactNative},
		{name: "amp", sourceType: "amp", want: common.SourceTypeAMP},
		{name: "flutter", sourceType: "flutter", want: common.SourceTypeFlutter},
		{name: "cordova", sourceType: "cordova", want: common.SourceTypeCordova},
		{name: "shopify", sourceType: "shopify", want: common.SourceTypeShopify},
		{name: "event-stream kotlin spelling", sourceType: "kotlin", want: common.SourceTypeAndroidKotlin},
		{name: "event-stream swift spelling", sourceType: "swift", want: common.SourceTypeIOSSwift},
		{name: "server sdk falls back to cloud", sourceType: "java", want: common.SourceTypeCloud},
		{name: "webhook falls back to cloud", sourceType: "webhook", category: "webhook", want: common.SourceTypeCloud},
		{name: "empty type falls back to cloud", sourceType: "", want: common.SourceTypeCloud},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, common.SourceTypeToken(c.sourceType, c.category))
		})
	}
}
