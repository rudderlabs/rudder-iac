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
