package definitions_test

import (
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type describeHeader struct {
	From string `mapstructure:"from" validate:"required,dynamic_or_pattern=describe_short"`
	To   string `mapstructure:"to" validate:"omitempty,pattern=describe_short"`
}

type describeFiltering struct {
	Allow []string `mapstructure:"allow" validate:"omitempty,excluded_with=Deny,dive,dynamic_or_pattern=describe_short"`
	Deny  []string `mapstructure:"deny" validate:"omitempty,excluded_with=Allow,dive,dynamic_or_pattern=describe_short"`
}

type describeSDK struct {
	Web *int `mapstructure:"web" validate:"omitempty,oneof=1 2" default:"2"`
}

type describeAuth struct {
	Token string `mapstructure:"token" validate:"required,describe_token_check"`
}

type describeConfig struct {
	APIKey            string                   `mapstructure:"api_key" validate:"required,dynamic_or_pattern=describe_short"`
	Region            string                   `mapstructure:"region" validate:"required,oneof=us eu"`
	Mode              *string                  `mapstructure:"mode" validate:"omitempty,dynamic_or_oneof=fast safe"`
	UseAuth           *bool                    `mapstructure:"use_auth" default:"false"`
	AuthKey           string                   `mapstructure:"auth_key" validate:"required_if=UseAuth true Region eu,max=20"`
	Fallback          string                   `mapstructure:"fallback" validate:"required_unless=Region us,required_with=AuthKey"`
	Prefix            string                   `mapstructure:"prefix" validate:"omitempty,min=2,max=8"`
	LegacyToken       string                   `mapstructure:"legacy_token" validate:"excluded_if=UseAuth false Region us"`
	Headers           []describeHeader         `mapstructure:"headers" validate:"omitempty,dive"`
	Filtering         *describeFiltering       `mapstructure:"filtering"`
	SDKVersion        *describeSDK             `mapstructure:"sdk_version"`
	Auth              *describeAuth            `mapstructure:"auth" validate:"required_if=UseAuth true"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

func describeDefinition() *definitions.DestinationDefinition {
	funcs.NewPatternWithReject("describe_short", `^.{0,10}$`, `^bad`, "must be short")

	return &definitions.DestinationDefinition{
		Type:        "describe",
		DisplayName: "Describe Test",
		APIType:     "DESCRIBE",
		Version:     1,
		Properties: []converter.ConfigProperty{
			converter.Gated(converter.Simple("sdkVersion.web", "sdk_version.web"), common.SourceTypeWeb),
		},
		SecretKeys: []string{"api_key", "headers.to"},
		NewConfig:  func() any { return &describeConfig{} },
		SourceTypes: []string{
			common.SourceTypeWeb,
			common.SourceTypeCloud,
		},
		ConnectionModes: map[string][]string{
			common.SourceTypeWeb:   {"cloud", "device"},
			common.SourceTypeCloud: {"cloud"},
		},
		ConnectionRequiredKeys: map[string]map[string][]string{
			common.SourceTypeWeb: {"device": {"region"}},
		},
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "describe_token_check", Func: func(validator.FieldLevel) bool { return true }},
		},
	}
}

func intPtr(v int) *int { return &v }

func TestDescribeReadsTheConfigModel(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(describeDefinition()))
	registered, err := registry.Get("describe", 1)
	require.NoError(t, err)

	assert.Equal(t, definitions.Description{
		Type:        "describe",
		DisplayName: "Describe Test",
		APIType:     "DESCRIBE",
		Version:     1,
		SourceTypes: []string{"web", "cloud"},
		ConnectionModes: map[string][]string{
			"web":   {"cloud", "device"},
			"cloud": {"cloud"},
		},
		ConnectionRequiredKeys: map[string]map[string][]string{
			"web": {"device": {"region"}},
		},
		SecretKeys: []string{"api_key", "headers.to"},
		Fields: []definitions.Field{
			{Key: "api_key", Type: "string", Required: true, Pattern: "describe_short", AcceptsTemplate: true, Secret: true},
			{Key: "region", Type: "string", Required: true, Enum: []any{"us", "eu"}},
			{Key: "mode", Type: "string", Enum: []any{"fast", "safe"}, OpenEnum: true, AcceptsTemplate: true},
			{Key: "use_auth", Type: "boolean", Default: false, HasDefault: true},
			{
				Key:  "auth_key",
				Type: "string",
				RequiredIf: []definitions.Condition{
					{Key: "use_auth", Value: true},
					{Key: "region", Value: "eu"},
				},
				Max: intPtr(20),
			},
			{
				Key:            "fallback",
				Type:           "string",
				RequiredUnless: []definitions.Condition{{Key: "region", Value: "us"}},
				RequiredWith:   []string{"auth_key"},
			},
			{Key: "prefix", Type: "string", Min: intPtr(2), Max: intPtr(8)},
			{
				Key:  "legacy_token",
				Type: "string",
				ExcludedIf: []definitions.Condition{
					{Key: "use_auth", Value: false},
					{Key: "region", Value: "us"},
				},
			},
			{Key: "headers[].from", Type: "string", Required: true, Pattern: "describe_short", AcceptsTemplate: true},
			{Key: "headers[].to", Type: "string", Pattern: "describe_short", Secret: true},
			{
				Key: "filtering.allow", Type: "array<string>", ExcludedWith: []string{"filtering.deny"},
				Pattern: "describe_short", AcceptsTemplate: true,
			},
			{
				Key: "filtering.deny", Type: "array<string>", ExcludedWith: []string{"filtering.allow"},
				Pattern: "describe_short", AcceptsTemplate: true,
			},
			{
				Key: "sdk_version.web", Type: "integer", Enum: []any{int64(1), int64(2)},
				Default: int64(2), HasDefault: true, OnlyForSourceTypes: []string{"web"},
			},
			{Key: "auth", Type: "object", RequiredIf: []definitions.Condition{{Key: "use_auth", Value: true}}},
			{Key: "auth.token", Type: "string", Required: true, CheckedBy: []string{"describe_token_check"}},
			{Key: "connection_mode", Type: "map<sourceToken,string>"},
			{Key: "consent_management", Type: "map<sourceToken,array<object>>"},
		},
		Patterns: map[string]funcs.PatternInfo{
			"describe_short": {Regex: `^.{0,10}$`, Reject: `^bad`, Message: "must be short"},
		},
		CustomValidators: []string{"describe_token_check"},
	}, registered.Describe())
}

func TestDescribeReturnsACopy(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(describeDefinition()))
	registered, err := registry.Get("describe", 1)
	require.NoError(t, err)

	first := registered.Describe()
	first.Fields[1].Enum[0] = "changed"
	first.ConnectionModes["web"][0] = "changed"

	second := registered.Describe()
	assert.Equal(t, []any{"us", "eu"}, second.Fields[1].Enum)
	assert.Equal(t, []string{"cloud", "device"}, second.ConnectionModes["web"])
}

func TestRegisterRequiresDisplayName(t *testing.T) {
	t.Parallel()

	def := describeDefinition()
	def.DisplayName = ""

	err := definitions.NewRegistry().Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination definition describe version 1 has no display name")
}

func TestRegisterRequiresSecretKeysToResolveToStrings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		secretKeys []string
		wantErr    string
	}{
		{name: "through a list of objects", secretKeys: []string{"headers.to"}},
		{name: "unknown key", secretKeys: []string{"api_token"}, wantErr: `secret key "api_token": "api_token" is not a config model field`},
		{name: "unknown nested key", secretKeys: []string{"headers.value"}, wantErr: `secret key "headers.value": "value" is not a config model field`},
		{name: "not a string", secretKeys: []string{"use_auth"}, wantErr: `secret key "use_auth": resolves to a bool field, not a string`},
		{name: "a container", secretKeys: []string{"auth"}, wantErr: `secret key "auth": resolves to a struct field, not a string`},
		{name: "into a list of strings", secretKeys: []string{"filtering.allow.x"}, wantErr: `"x" is not inside an object or a list of objects`},
		{name: "into a map", secretKeys: []string{"connection_mode.web"}, wantErr: `"web" is not inside an object or a list of objects`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			def := describeDefinition()
			def.SecretKeys = tc.secretKeys

			err := definitions.NewRegistry().Register(def)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestRegisterRejectsValidateRulesDescribeCannotRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		newConfig func() any
		wantErr   string
	}{
		{
			name: "a rule neither described nor registered",
			newConfig: func() any {
				return &struct {
					Name string `mapstructure:"name" validate:"omitempty,len=3"`
				}{}
			},
			wantErr: `config key "name": validate rule "len" is neither described nor one of the definition's ConfigValidateFuncs`,
		},
		{
			name: "an unregistered pattern",
			newConfig: func() any {
				return &struct {
					Name string `mapstructure:"name" validate:"omitempty,dynamic_or_pattern=describe_missing"`
				}{}
			},
			wantErr: `config key "name": dynamic_or_pattern: pattern "describe_missing" is not registered`,
		},
		{
			name: "a condition on an unknown field",
			newConfig: func() any {
				return &struct {
					Name string `mapstructure:"name" validate:"required_if=Missing true"`
				}{}
			},
			wantErr: `config key "name": required_if: field "Missing" does not resolve to a config model field`,
		},
		{
			name: "a condition value the field cannot hold",
			newConfig: func() any {
				return &struct {
					Flag *bool  `mapstructure:"flag"`
					Name string `mapstructure:"name" validate:"required_if=Flag yes"`
				}{}
			},
			wantErr: `config key "name": required_if: value for flag: invalid bool "yes"`,
		},
		{
			name: "an enum value the field cannot hold",
			newConfig: func() any {
				return &struct {
					Size *int `mapstructure:"size" validate:"omitempty,oneof=1 big"`
				}{}
			},
			wantErr: `config key "size": oneof: invalid integer "big"`,
		},
		{
			name: "required on each item of a list",
			newConfig: func() any {
				return &struct {
					Tags []string `mapstructure:"tags" validate:"omitempty,dive,required"`
				}{}
			},
			wantErr: `config key "tags": validate rule "required" after dive would be described as a rule on the list`,
		},
		{
			name: "max on each item of a list",
			newConfig: func() any {
				return &struct {
					Tags []string `mapstructure:"tags" validate:"omitempty,dive,max=5"`
				}{}
			},
			wantErr: `config key "tags": validate rule "max" after dive would be described as a rule on the list`,
		},
		{
			name: "required inside a block that is not a pointer",
			newConfig: func() any {
				return &struct {
					Inner struct {
						Must string `mapstructure:"must" validate:"required"`
					} `mapstructure:"inner"`
				}{}
			},
			wantErr: `config key "inner.must": required inside a block that is not a pointer applies even when the spec leaves the block out`,
		},
		{
			name: "required on a bool that is not a pointer",
			newConfig: func() any {
				return &struct {
					Flag bool `mapstructure:"flag" validate:"required"`
				}{}
			},
			wantErr: `config key "flag": a required rule on a bool that is not a pointer rejects false`,
		},
		{
			name: "required_if on a number that is not a pointer",
			newConfig: func() any {
				return &struct {
					Flag *bool `mapstructure:"flag"`
					Size int   `mapstructure:"size" validate:"required_if=Flag true"`
				}{}
			},
			wantErr: `config key "size": a required rule on a int that is not a pointer rejects 0`,
		},
		{
			name: "an or rule",
			newConfig: func() any {
				return &struct {
					Mode string `mapstructure:"mode" validate:"omitempty,oneof=a b|eq=c"`
				}{}
			},
			wantErr: `config key "mode": validate rule "oneof=a b|eq=c" uses |, a quote or an escaped character`,
		},
		{
			name: "a quoted parameter",
			newConfig: func() any {
				return &struct {
					Event string `mapstructure:"event" validate:"omitempty,oneof='Page View' Track"`
				}{}
			},
			wantErr: `config key "event": validate rule "oneof='Page View' Track" uses |, a quote or an escaped character`,
		},
		{
			name: "item rules on a list of objects",
			newConfig: func() any {
				return &struct {
					Items []struct {
						Name string `mapstructure:"name"`
					} `mapstructure:"items" validate:"dive,required"`
				}{}
			},
			wantErr: `config key "items": rules after dive apply only to a list of scalars`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := definitions.NewRegistry().Register(&definitions.DestinationDefinition{
				Type:        "TEST",
				DisplayName: "Test",
				Version:     1,
				NewConfig:   tc.newConfig,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestRegistryDefinitionsSortsByTypeAndVersion(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(definitions.WebhookTestDefinition("WEBHOOK", 2)))
	require.NoError(t, registry.Register(definitions.WebhookTestDefinition("WEBHOOK", 1)))
	require.NoError(t, registry.Register(definitions.GA4TestDefinition()))

	var got []string
	for _, def := range registry.Definitions() {
		got = append(got, fmt.Sprintf("%s@%d", def.Type, def.Version))
	}
	assert.Equal(t, []string{"GA4@1", "WEBHOOK@1", "WEBHOOK@2"}, got)
}
