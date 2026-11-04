package customaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	// Byte-identical to the real constraint behind webhook's webhookUrl, but
	// registered under its own name rather than reusing webhook_url: that name
	// exists only once the webhook package is linked, and coupling this
	// definition's validity to that import would break a standalone registry.
	baseURLPattern = `^(https?://)([a-zA-Z0-9-]{1,63}\.)+[a-zA-Z]{2,}(:(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5]\d{4}|[1-9]\d{1,3}))?(/.*)?$`
	// RE2 has no lookahead, so schema.json's two negative lookaheads become a
	// reject pattern. Both are anchored to the position just after the scheme,
	// which is what makes this equivalent rather than merely similar: ngrok is
	// rejected only as the first label, while `.localhost` is rejected anywhere.
	baseURLRejectPattern = `^https?://([a-zA-Z0-9-]*\.ngrok\.io|localhost|.*\.localhost)`
)

func init() {
	funcs.NewPatternWithReject(
		"custom_audience_base_url",
		baseURLPattern,
		baseURLRejectPattern,
		"must be a public http(s) domain URL and must not use localhost or ngrok",
	)
}

// Source types from integrations-config destinations/custom_audience
// db-config.json: a warehouse-only destination, reached through rETL alone.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// customAudienceConfig is the local YAML config model. The field set mirrors
// integrations-config destinations/custom_audience defaultConfig; validation
// mirrors schema.json.
//
// This is the only vdm-next destination whose config is not flat: `actions`
// describes the three HTTP calls the sync makes, so it nests exactly as the
// upstream payload does.
//
// consent_management is absent: db-config declares no per-source-type
// destConfig lists at all, so the destination has no consent surface.
type customAudienceConfig struct {
	// Required only once authentication_enabled is true (schema.json allOf):
	// an unauthenticated endpoint needs no account.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required_if=AuthenticationEnabled true,omitempty,pattern=single_line_100"`
	// The endpoint the sync posts to. Must be a public http(s) domain:
	// localhost and ngrok hosts are rejected, as upstream rejects them.
	BaseURL string `mapstructure:"base_url" validate:"required,pattern=custom_audience_base_url"`
	// A pointer so that absent stays distinct from false, which is what
	// rudder_account_id's condition turns on.
	AuthenticationEnabled *bool `mapstructure:"authentication_enabled"`
	// Extra request headers. Same shape on both sides, so no field rename.
	Headers []header `mapstructure:"headers" validate:"omitempty,dive"`
	// The insert, update and delete calls. All three are required upstream.
	Actions *actions `mapstructure:"actions" validate:"required"`
	// Credentials for the chosen auth scheme. All five sit in db-config
	// defaultConfig with no schema.json entry, so they carry no constraint
	// beyond the secrecy of three of them; which ones apply depends on the
	// auth type configured on the linked account.
	BasicAuthUserName string                `mapstructure:"basic_auth_user_name"`
	BasicAuthPassword string                `mapstructure:"basic_auth_password"`
	BearerToken       string                `mapstructure:"bearer_token"`
	APIKeyName        string                `mapstructure:"api_key_name"`
	APIKeyValue       string                `mapstructure:"api_key_value"`
	ConnectionMode    common.ConnectionMode `mapstructure:"connection_mode"`
}

type header struct {
	Key   string `mapstructure:"key" validate:"required"`
	Value string `mapstructure:"value" validate:"required"`
}

// actions holds the three calls. Each is a pointer so `required` reports a
// missing block rather than reading an empty struct as present.
type actions struct {
	Insert *actionConfig `mapstructure:"insert" validate:"required"`
	Update *updateAction `mapstructure:"update" validate:"required"`
	Delete *actionConfig `mapstructure:"delete" validate:"required"`
}

// actionConfig is schema.json's baseActionConfig, used by insert and delete.
type actionConfig struct {
	RequestBody string `mapstructure:"request_body" validate:"required"`
	Method      string `mapstructure:"method" validate:"required,oneof=POST PUT PATCH GET DELETE"`
	Endpoint    string `mapstructure:"endpoint" validate:"required"`
	// A pointer because `required` on a bare int would reject a 0 the enum
	// below already rules out, and absence must be reported as absence.
	BatchSize *int    `mapstructure:"batch_size" validate:"required,min=1,max=5000"`
	Fields    []field `mapstructure:"fields" validate:"omitempty,dive"`
}

// updateAction is baseActionConfig plus use_insert_config, which exempts the
// four otherwise-required keys by reusing the insert call's shape. schema.json
// states this as an if/then/else on the update block; required_unless is the
// same rule read from the other side.
type updateAction struct {
	UseInsertConfig *bool   `mapstructure:"use_insert_config"`
	RequestBody     string  `mapstructure:"request_body" validate:"required_unless=UseInsertConfig true"`
	Method          string  `mapstructure:"method" validate:"required_unless=UseInsertConfig true,omitempty,oneof=POST PUT PATCH GET DELETE"`
	Endpoint        string  `mapstructure:"endpoint" validate:"required_unless=UseInsertConfig true"`
	BatchSize       *int    `mapstructure:"batch_size" validate:"required_unless=UseInsertConfig true,omitempty,min=1,max=5000"`
	Fields          []field `mapstructure:"fields" validate:"omitempty,dive"`
}

// field is one column the call sends, and how to hash it.
type field struct {
	Name       string `mapstructure:"name" validate:"required"`
	IsRequired *bool  `mapstructure:"is_required" validate:"required"`
	HashType   string `mapstructure:"hash_type" validate:"required,oneof=NONE SHA256 SHA512 MD5"`
	IsCustom   *bool  `mapstructure:"is_custom" validate:"required"`
}

// actionProperties maps one action's keys. The API nests the same way the local
// spec does, so both sides are dotted paths differing only in key casing.
func actionProperties(action string) []converter.ConfigProperty {
	var (
		apiRoot   = "actions." + action
		localRoot = "actions." + action
	)
	return []converter.ConfigProperty{
		converter.Simple(apiRoot+".requestBody", localRoot+".request_body"),
		converter.Simple(apiRoot+".method", localRoot+".method"),
		converter.Simple(apiRoot+".endpoint", localRoot+".endpoint"),
		converter.Simple(apiRoot+".batchSize", localRoot+".batch_size"),
		converter.ArrayWithObjects(apiRoot+".fields", localRoot+".fields", map[string]any{
			"name":       "name",
			"isRequired": "is_required",
			"hashType":   "hash_type",
			"isCustom":   "is_custom",
		}),
	}
}

// NewDefinition returns the Custom Activation destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("baseUrl", "base_url"),
		converter.Simple("authenticationEnabled", "authentication_enabled"),
		converter.Simple("headers", "headers"),
		converter.Simple("basicAuthUserName", "basic_auth_user_name"),
		converter.Simple("basicAuthPassword", "basic_auth_password"),
		converter.Simple("bearerToken", "bearer_token"),
		converter.Simple("apiKeyName", "api_key_name"),
		converter.Simple("apiKeyValue", "api_key_value"),
	}
	properties = append(properties, actionProperties("insert")...)
	properties = append(properties, actionProperties("delete")...)
	properties = append(properties, actionProperties("update")...)
	properties = append(properties, converter.Simple("actions.update.useInsertConfig", "actions.update.use_insert_config"))
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "custom_audience",
		DisplayName: "Custom Activation",
		APIType:     "CUSTOM_AUDIENCE",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{"basic_auth_password", "bearer_token", "api_key_value"},
		NewConfig: func() any {
			return &customAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
