package definitions_test

import (
	"fmt"
	"strings"
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
	Mode              *string                  `mapstructure:"mode" validate:"omitempty,dynamic_or_oneof='' fast safe"`
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

// oneof and dynamic_or_oneof read members the way go-playground does: quotes
// group a member that holds spaces, and a pair with nothing between them is "".
func TestDescribeReadsQuotedOneOfMembers(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "quoted",
		DisplayName: "Quoted",
		Version:     1,
		NewConfig: func() any {
			return &struct {
				Middle string `mapstructure:"middle" validate:"omitempty,oneof=a '' b"`
				End    string `mapstructure:"end" validate:"omitempty,dynamic_or_oneof=a b ''"`
				Alone  string `mapstructure:"alone" validate:"omitempty,oneof=''"`
				Spaced string `mapstructure:"spaced" validate:"omitempty,dynamic_or_oneof='Page View' Track"`
			}{}
		},
	}))
	registered, err := registry.Get("quoted", 1)
	require.NoError(t, err)

	enums := map[string][]any{}
	for _, field := range registered.Describe().Fields {
		enums[field.Key] = field.Enum
	}
	assert.Equal(t, map[string][]any{
		"middle": {"a", "", "b"},
		"end":    {"a", "b", ""},
		"alone":  {""},
		"spaced": {"Page View", "Track"},
	}, enums)

	assert.Empty(t, registered.ValidateConfig(map[string]any{"middle": "", "end": "", "alone": "", "spaced": "Page View"}))
	want := []definitions.ConfigError{{
		Path:    "/spaced",
		Message: `'spaced' must be one of ["Page View" Track] or a dynamic config value (env.VAR, {{ path || fallback }}, or {{ .VAR }})`,
	}}
	assert.Equal(t, want, registered.ValidateConfig(map[string]any{"spaced": "Page"}))
	assert.Equal(t, want, registered.ValidateConfig(map[string]any{"spaced": ""}))
}

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
			{Key: "mode", Type: "string", Enum: []any{"", "fast", "safe"}, OpenEnum: true, AcceptsTemplate: true},
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
			{
				Key: "consent_management.<sourceToken>[].provider", Type: "string", Required: true,
				Enum: []any{"custom", "iubenda", "ketch", "oneTrust"}, Unique: true,
			},
			{
				Key:        "consent_management.<sourceToken>[].resolution_strategy",
				Type:       "string",
				RequiredIf: []definitions.Condition{{Key: "consent_management.<sourceToken>[].provider", Value: "custom"}},
				Enum:       []any{"and", "or"},
			},
			{Key: "consent_management.<sourceToken>[].consents", Type: "array<string>", Pattern: "consent_value", AcceptsTemplate: true},
		},
		Patterns: map[string]funcs.PatternInfo{
			"describe_short": {Regex: `^.{0,10}$`, Reject: `^bad`, Message: "must be short"},
			"consent_value":  {Regex: `^(\{\{.*\|\|.*\}\}|env\..+|.{0,100})$`, Message: "must be at most 100 characters"},
		},
		CustomValidators: []string{"describe_token_check"},
	}, registered.Describe())
}

// A definition that replaces the canonical consent check for any source token
// documents no consent entry rules, since only the override's code says what
// it accepts. Narrowing them to the other tokens would read, under --source,
// as if the overridden token took no consent entries.
func TestDescribeLeavesOutConsentEntriesWhenOverridden(t *testing.T) {
	t.Parallel()

	override := func([]common.ConsentEntry) []common.ConsentValidationError { return nil }
	cases := []struct {
		name      string
		overrides map[string]common.ConsentValidator
		want      []string
	}{
		{
			name: "no override",
			want: []string{
				"consent_management.<sourceToken>[].provider",
				"consent_management.<sourceToken>[].resolution_strategy",
				"consent_management.<sourceToken>[].consents",
			},
		},
		{
			name:      "an override on one source token",
			overrides: map[string]common.ConsentValidator{"web": override},
		},
		{
			name:      "an override on every source token",
			overrides: map[string]common.ConsentValidator{"web": override, "cloud": override},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			def := describeDefinition()
			def.ConsentValidationOverrides = tc.overrides
			registry := definitions.NewRegistry()
			require.NoError(t, registry.Register(def))
			registered, err := registry.Get("describe", 1)
			require.NoError(t, err)
			desc := registered.Describe()

			var entryKeys []string
			for _, f := range desc.Fields {
				if strings.HasPrefix(f.Key, "consent_management.") {
					entryKeys = append(entryKeys, f.Key)
				}
			}
			assert.Equal(t, tc.want, entryKeys)
			_, hasPattern := desc.Patterns[common.ConsentValuePattern]
			assert.Equal(t, len(tc.want) > 0, hasPattern, "the consent_value pattern")
		})
	}
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

// An integer oneof or condition value is read as 64 bits on every platform,
// so a value above 2147483647 also registers on the 32-bit release builds.
// Run it there with GOARCH=386.
func TestDescribeReadsInt64RuleValues(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		DisplayName: "Test",
		Version:     1,
		NewConfig: func() any {
			return &struct {
				Size *int64  `mapstructure:"size" validate:"omitempty,oneof=3000000000 9223372036854775807"`
				Name *string `mapstructure:"name" validate:"required_if=Size 3000000000"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	assert.Equal(t, []definitions.Field{
		{Key: "size", Type: "integer", Enum: []any{int64(3000000000), int64(9223372036854775807)}},
		{Key: "name", Type: "string", RequiredIf: []definitions.Condition{{Key: "size", Value: int64(3000000000)}}},
	}, registered.Describe().Fields)
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
		{name: "unknown key", secretKeys: []string{"api_token"}, wantErr: `secret key "api_token" does not name a string field of the config model`},
		{name: "unknown nested key", secretKeys: []string{"headers.value"}, wantErr: `secret key "headers.value" does not name a string field of the config model`},
		{name: "not a string", secretKeys: []string{"use_auth"}, wantErr: `secret key "use_auth" names a field of type boolean, not string`},
		{name: "a list of strings", secretKeys: []string{"filtering.allow"}, wantErr: `secret key "filtering.allow" names a field of type array<string>, not string`},
		{name: "a container with a rule", secretKeys: []string{"auth"}, wantErr: `secret key "auth" names a field of type object, not string`},
		{name: "a container without a rule", secretKeys: []string{"filtering"}, wantErr: `secret key "filtering" does not name a string field of the config model`},
		{name: "into a list of strings", secretKeys: []string{"filtering.allow.x"}, wantErr: `secret key "filtering.allow.x" does not name a string field of the config model`},
		{name: "into a map", secretKeys: []string{"connection_mode.web"}, wantErr: `secret key "connection_mode.web" does not name a string field of the config model`},
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

// A JSONPath secret key exempts a secret key from validate's secret-reference
// warning, so it must be one.
func TestRegisterRequiresJSONPathSecretKeysToBeSecretKeys(t *testing.T) {
	t.Parallel()

	def := describeDefinition()
	def.SecretKeys = []string{"headers.to"}
	def.JSONPathSecretKeys = []string{"headers.to"}
	require.NoError(t, definitions.NewRegistry().Register(def))

	def = describeDefinition()
	def.SecretKeys = []string{"headers.to"}
	def.JSONPathSecretKeys = []string{"api_key"}
	err := definitions.NewRegistry().Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `JSONPath secret key "api_key" is not a secret key`)
}

// The docs check only the spec ID when they name a secret's variable, which
// holds because every key segment is snake_case.
func TestRegisterRequiresSnakeCaseSecretKeys(t *testing.T) {
	t.Parallel()

	err := definitions.NewRegistry().Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		DisplayName: "Test",
		Version:     1,
		SecretKeys:  []string{"client-secret"},
		NewConfig: func() any {
			return &struct {
				ClientSecret string `mapstructure:"client-secret"`
			}{}
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `secret key "client-secret" is not snake_case`)
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
			wantErr: `reading config key "name": validate rule "len" is neither described nor one of the definition's ConfigValidateFuncs`,
		},
		{
			name: "an unregistered pattern",
			newConfig: func() any {
				return &struct {
					Name string `mapstructure:"name" validate:"omitempty,dynamic_or_pattern=describe_missing"`
				}{}
			},
			wantErr: `reading config key "name": reading dynamic_or_pattern rule: pattern "describe_missing" is not registered`,
		},
		{
			name: "a condition on an unknown field",
			newConfig: func() any {
				return &struct {
					Name string `mapstructure:"name" validate:"required_if=Missing true"`
				}{}
			},
			wantErr: `reading config key "name": reading required_if rule: field "Missing" does not resolve to a config model field`,
		},
		{
			name: "a condition value the field cannot hold",
			newConfig: func() any {
				return &struct {
					Flag *bool  `mapstructure:"flag"`
					Name string `mapstructure:"name" validate:"required_if=Flag yes"`
				}{}
			},
			wantErr: `reading config key "name": reading required_if rule: parsing the value for flag: want true or false, got "yes"`,
		},
		{
			name: "an enum value the field cannot hold",
			newConfig: func() any {
				return &struct {
					Size *int `mapstructure:"size" validate:"omitempty,oneof=1 big"`
				}{}
			},
			wantErr: `reading config key "size": reading oneof rule: want a plain base-10 integer, got "big"`,
		},
		{
			name: "required on each item of a list",
			newConfig: func() any {
				return &struct {
					Tags []string `mapstructure:"tags" validate:"omitempty,dive,required"`
				}{}
			},
			wantErr: `reading config key "tags": validate rule "required" after dive would be described as a rule on the list`,
		},
		{
			name: "max on each item of a list",
			newConfig: func() any {
				return &struct {
					Tags []string `mapstructure:"tags" validate:"omitempty,dive,max=5"`
				}{}
			},
			wantErr: `reading config key "tags": validate rule "max" after dive would be described as a rule on the list`,
		},
		{
			name: "omitempty next to an enum on each item of a list",
			newConfig: func() any {
				return &struct {
					Tags []string `mapstructure:"tags" validate:"omitempty,dive,omitempty,oneof=a b"`
				}{}
			},
			wantErr: `reading config key "tags": omitempty after dive skips the enum on an empty item`,
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
			wantErr: `reading config key "inner.must": required inside a block that is not a pointer applies even when the spec leaves the block out`,
		},
		{
			name: "required on a bool that is not a pointer",
			newConfig: func() any {
				return &struct {
					Flag bool `mapstructure:"flag" validate:"required"`
				}{}
			},
			wantErr: `reading config key "flag": a required rule on a bool that is not a pointer rejects false`,
		},
		{
			name: "required_if on a number that is not a pointer",
			newConfig: func() any {
				return &struct {
					Flag *bool `mapstructure:"flag"`
					Size int   `mapstructure:"size" validate:"required_if=Flag true"`
				}{}
			},
			wantErr: `reading config key "size": a required rule on a int that is not a pointer rejects 0`,
		},
		{
			name: "required_with after omitempty",
			newConfig: func() any {
				return &struct {
					Other string `mapstructure:"other"`
					Name  string `mapstructure:"name" validate:"omitempty,required_with=Other"`
				}{}
			},
			wantErr: `reading config key "name": validate rule "required_with" comes after omitempty, which skips it on an empty value`,
		},
		{
			name: "required_if after omitempty",
			newConfig: func() any {
				return &struct {
					Flag *bool  `mapstructure:"flag"`
					Name string `mapstructure:"name" validate:"omitempty,max=5,required_if=Flag true"`
				}{}
			},
			wantErr: `reading config key "name": validate rule "required_if" comes after omitempty, which skips it on an empty value`,
		},
		{
			name: "a bool condition value go-playground reads as false",
			newConfig: func() any {
				return &struct {
					Flag *bool  `mapstructure:"flag"`
					Name string `mapstructure:"name" validate:"required_if=Flag True"`
				}{}
			},
			wantErr: `reading config key "name": reading required_if rule: parsing the value for flag: want true or false, got "True"`,
		},
		{
			name: "an integer condition value go-playground reads as octal",
			newConfig: func() any {
				return &struct {
					Level *int   `mapstructure:"level"`
					Name  string `mapstructure:"name" validate:"required_unless=Level 010"`
				}{}
			},
			wantErr: `reading config key "name": reading required_unless rule: parsing the value for level: want a plain base-10 integer, got "010"`,
		},
		{
			name: "an integer enum value oneof never matches",
			newConfig: func() any {
				return &struct {
					Size *int `mapstructure:"size" validate:"omitempty,oneof=+1 2"`
				}{}
			},
			wantErr: `reading config key "size": reading oneof rule: want a plain base-10 integer, got "+1"`,
		},
		{
			name: "a condition on a list",
			newConfig: func() any {
				return &struct {
					Tags []string `mapstructure:"tags"`
					Name string   `mapstructure:"name" validate:"required_if=Tags 2"`
				}{}
			},
			wantErr: `reading config key "name": reading required_if rule: parsing the value for tags: cannot compare a slice with "2"`,
		},
		{
			name: "required on a block that is not a pointer",
			newConfig: func() any {
				return &struct {
					Inner struct {
						Name string `mapstructure:"name"`
					} `mapstructure:"inner" validate:"required"`
				}{}
			},
			wantErr: `reading config key "inner": a required rule on a block that is not a pointer is skipped or rejects an empty block`,
		},
		{
			name: "required_if on a block that is not a pointer",
			newConfig: func() any {
				return &struct {
					Flag  *bool `mapstructure:"flag"`
					Inner struct {
						Name string `mapstructure:"name"`
					} `mapstructure:"inner" validate:"required_if=Flag true"`
				}{}
			},
			wantErr: `reading config key "inner": a required rule on a block that is not a pointer is skipped or rejects an empty block`,
		},
		{
			name: "an or rule",
			newConfig: func() any {
				return &struct {
					Mode string `mapstructure:"mode" validate:"omitempty,oneof=a b|eq=c"`
				}{}
			},
			wantErr: `reading config key "mode": validate rule "oneof=a b|eq=c" uses |, a quote or an escaped character`,
		},
		{
			name: "a quoted parameter",
			newConfig: func() any {
				return &struct {
					Event string `mapstructure:"event" validate:"omitempty,excluded_if=Mode 'fast mode'"`
					Mode  string `mapstructure:"mode"`
				}{}
			},
			wantErr: `reading config key "event": validate rule "excluded_if=Mode 'fast mode'" uses |, a quote or an escaped character`,
		},
		{
			name: "an escaped comma",
			newConfig: func() any {
				return &struct {
					Mode string `mapstructure:"mode" validate:"omitempty,oneof=a0x2Cb c"`
				}{}
			},
			wantErr: `reading config key "mode": validate rule "oneof=a0x2Cb c" uses |, a quote or an escaped character`,
		},
		{
			name: "an escaped pipe",
			newConfig: func() any {
				return &struct {
					Mode string `mapstructure:"mode" validate:"omitempty,oneof=a0x7Cb c"`
				}{}
			},
			wantErr: `reading config key "mode": validate rule "oneof=a0x7Cb c" uses |, a quote or an escaped character`,
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
			wantErr: `reading config key "items": rules after dive apply only to a list of scalars`,
		},
		{
			name: "oneof on a bool",
			newConfig: func() any {
				return &struct {
					Flag *bool `mapstructure:"flag" validate:"omitempty,oneof=true false"`
				}{}
			},
			wantErr: `reading config key "flag": validate rule "oneof" on a bool makes go-playground panic`,
		},
		{
			name: "oneof on a number",
			newConfig: func() any {
				return &struct {
					Rate *float64 `mapstructure:"rate" validate:"omitempty,oneof=1.5 2.5"`
				}{}
			},
			wantErr: `reading config key "rate": validate rule "oneof" on a float64 makes go-playground panic`,
		},
		{
			name: "oneof on a list without dive",
			newConfig: func() any {
				return &struct {
					Tags []string `mapstructure:"tags" validate:"omitempty,oneof=a b"`
				}{}
			},
			wantErr: `reading config key "tags": validate rule "oneof" on a slice makes go-playground panic`,
		},
		{
			name: "dynamic_or_oneof on an integer",
			newConfig: func() any {
				return &struct {
					Size *int `mapstructure:"size" validate:"omitempty,dynamic_or_oneof=1 2"`
				}{}
			},
			wantErr: `reading config key "size": validate rule "dynamic_or_oneof" on a int does not check the value`,
		},
		{
			name: "pattern on an integer",
			newConfig: func() any {
				return &struct {
					Size *int `mapstructure:"size" validate:"omitempty,pattern=describe_missing"`
				}{}
			},
			wantErr: `reading config key "size": validate rule "pattern" on a int does not check the value`,
		},
		{
			name: "dynamic_or_pattern on an integer",
			newConfig: func() any {
				return &struct {
					Size *int `mapstructure:"size" validate:"omitempty,dynamic_or_pattern=describe_short"`
				}{}
			},
			wantErr: `reading config key "size": validate rule "dynamic_or_pattern" on a int does not check the value`,
		},
		{
			name: "min on a bool",
			newConfig: func() any {
				return &struct {
					Flag *bool `mapstructure:"flag" validate:"omitempty,min=1"`
				}{}
			},
			wantErr: `reading config key "flag": validate rule "min" on a bool makes go-playground panic`,
		},
		{
			name: "max on a block",
			newConfig: func() any {
				return &struct {
					Inner *struct {
						Name string `mapstructure:"name"`
					} `mapstructure:"inner" validate:"omitempty,max=1"`
				}{}
			},
			wantErr: `reading config key "inner": validate rule "max" on a struct makes go-playground panic`,
		},
		{
			name: "a limit go-playground reads as octal",
			newConfig: func() any {
				return &struct {
					Name string `mapstructure:"name" validate:"omitempty,min=010"`
				}{}
			},
			wantErr: `reading config key "name": reading min rule: want a plain base-10 integer, got "010"`,
		},
		{
			name: "a negative limit on an unsigned integer",
			newConfig: func() any {
				return &struct {
					Count *uint `mapstructure:"count" validate:"omitempty,min=-1"`
				}{}
			},
			wantErr: `reading config key "count": reading min rule: want 0 or more for a uint, got "-1"`,
		},
		{
			name: "a negative condition value on an unsigned integer",
			newConfig: func() any {
				return &struct {
					Count *uint   `mapstructure:"count"`
					Name  *string `mapstructure:"name" validate:"required_if=Count -1"`
				}{}
			},
			wantErr: `reading config key "name": reading required_if rule: parsing the value for count: want 0 or more for a uint, got "-1"`,
		},
		{
			name: "two required_if rules",
			newConfig: func() any {
				return &struct {
					A    *string `mapstructure:"a"`
					B    *string `mapstructure:"b"`
					Name *string `mapstructure:"name" validate:"required_if=A x,required_if=B y"`
				}{}
			},
			wantErr: `reading config key "name": validate rule "required_if" appears twice on one key`,
		},
		{
			name: "two required_unless rules",
			newConfig: func() any {
				return &struct {
					A    *string `mapstructure:"a"`
					B    *string `mapstructure:"b"`
					Name *string `mapstructure:"name" validate:"required_unless=A x,required_unless=B y"`
				}{}
			},
			wantErr: `reading config key "name": validate rule "required_unless" appears twice on one key`,
		},
		{
			name: "two excluded_if rules",
			newConfig: func() any {
				return &struct {
					A    *string `mapstructure:"a"`
					B    *string `mapstructure:"b"`
					Name *string `mapstructure:"name" validate:"excluded_if=A x,excluded_if=B y"`
				}{}
			},
			wantErr: `reading config key "name": validate rule "excluded_if" appears twice on one key`,
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
