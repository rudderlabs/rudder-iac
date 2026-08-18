package confluentcloud

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/confluent_cloud/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeShopify,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeAMP:           {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeWarehouse:     {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

// Source types from schema.json oneTrustCookieCategories/ketchConsentPurposes.
var consentCategorySourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeIOS,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeShopify,
}

// confluentCloudConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/confluent_cloud schema/defaultConfig;
// validations mirror schema.json.
type confluentCloudConfig struct {
	BootstrapServer          string                     `mapstructure:"bootstrap_server" validate:"required,dynamic_or_pattern=single_line_100"`
	Topic                    string                     `mapstructure:"topic" validate:"required,dynamic_or_pattern=single_line_100"`
	APIKey                   string                     `mapstructure:"api_key" validate:"required,dynamic_or_pattern=single_line_100"`
	APISecret                string                     `mapstructure:"api_secret" validate:"required,dynamic_or_pattern=single_line_100"`
	ConsentManagement        common.ConsentManagement   `mapstructure:"consent_management"`
	OneTrustCookieCategories sourceTypeCookieCategories `mapstructure:"one_trust_cookie_categories"`
	KetchConsentPurposes     sourceTypeKetchPurposes    `mapstructure:"ketch_consent_purposes"`
}

type sourceTypeCookieCategories struct {
	Android     []oneTrustCookieCategory `mapstructure:"android" validate:"omitempty,dive"`
	IOS         []oneTrustCookieCategory `mapstructure:"ios" validate:"omitempty,dive"`
	Web         []oneTrustCookieCategory `mapstructure:"web" validate:"omitempty,dive"`
	Unity       []oneTrustCookieCategory `mapstructure:"unity" validate:"omitempty,dive"`
	AMP         []oneTrustCookieCategory `mapstructure:"amp" validate:"omitempty,dive"`
	Cloud       []oneTrustCookieCategory `mapstructure:"cloud" validate:"omitempty,dive"`
	Warehouse   []oneTrustCookieCategory `mapstructure:"warehouse" validate:"omitempty,dive"`
	ReactNative []oneTrustCookieCategory `mapstructure:"react_native" validate:"omitempty,dive"`
	Flutter     []oneTrustCookieCategory `mapstructure:"flutter" validate:"omitempty,dive"`
	Cordova     []oneTrustCookieCategory `mapstructure:"cordova" validate:"omitempty,dive"`
	Shopify     []oneTrustCookieCategory `mapstructure:"shopify" validate:"omitempty,dive"`
}

type oneTrustCookieCategory struct {
	OneTrustCookieCategory string `mapstructure:"one_trust_cookie_category" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type sourceTypeKetchPurposes struct {
	Android     []ketchConsentPurpose `mapstructure:"android" validate:"omitempty,dive"`
	IOS         []ketchConsentPurpose `mapstructure:"ios" validate:"omitempty,dive"`
	Web         []ketchConsentPurpose `mapstructure:"web" validate:"omitempty,dive"`
	Unity       []ketchConsentPurpose `mapstructure:"unity" validate:"omitempty,dive"`
	AMP         []ketchConsentPurpose `mapstructure:"amp" validate:"omitempty,dive"`
	Cloud       []ketchConsentPurpose `mapstructure:"cloud" validate:"omitempty,dive"`
	Warehouse   []ketchConsentPurpose `mapstructure:"warehouse" validate:"omitempty,dive"`
	ReactNative []ketchConsentPurpose `mapstructure:"react_native" validate:"omitempty,dive"`
	Flutter     []ketchConsentPurpose `mapstructure:"flutter" validate:"omitempty,dive"`
	Cordova     []ketchConsentPurpose `mapstructure:"cordova" validate:"omitempty,dive"`
	Shopify     []ketchConsentPurpose `mapstructure:"shopify" validate:"omitempty,dive"`
}

type ketchConsentPurpose struct {
	Purpose string `mapstructure:"purpose" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

// NewDefinition returns the Confluent Cloud destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("bootstrapServer", "bootstrap_server"),
		converter.Simple("topic", "topic"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("apiSecret", "api_secret"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)
	properties = append(properties, common.OneTrustCookieCategoryProperties(consentCategorySourceTypes)...)
	properties = append(properties, common.KetchConsentPurposeProperties(consentCategorySourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "confluent_cloud",
		APIType:    "CONFLUENT_CLOUD",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_secret", "api_key"},
		NewConfig: func() any {
			return &confluentCloudConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
