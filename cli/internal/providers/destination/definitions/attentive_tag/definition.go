package attentivetag

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// Strict digits-only — no env./{{ }} escape hatch (unlike terraform / schema.json).
	funcs.NewPattern(
		"attentive_sign_up_source_id",
		`^[0-9]*$`,
		"must contain only digits",
	)
}

// Source types from integrations-config destinations/attentive_tag/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeCloud,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

// attentiveTagConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/attentive_tag defaultConfig; validation
// constraints mirror overlapping schema.json rules.
type attentiveTagConfig struct {
	APIKey                string                   `mapstructure:"api_key" validate:"required,min=1,max=100"`
	SignUpSourceID        string                   `mapstructure:"sign_up_source_id" validate:"omitempty,pattern=attentive_sign_up_source_id"`
	EnableNewIdentifyFlow *bool                    `mapstructure:"enable_new_identify_flow"`
	ConsentManagement     common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Attentive Tag destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiKey", "api_key"),
		converter.Simple("signUpSourceId", "sign_up_source_id", converter.SkipZeroValue),
		converter.Simple("enableNewIdentifyFlow", "enable_new_identify_flow", converter.SkipZeroValue),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "attentive_tag",
		APIType:    "ATTENTIVE_TAG",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_key"},
		NewConfig: func() any {
			return &attentiveTagConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
