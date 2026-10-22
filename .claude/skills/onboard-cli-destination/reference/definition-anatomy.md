# Definition Anatomy — Annotated S3 Walkthrough

`cli/internal/providers/destination/definitions/s3/definition.go` is the
canonical template. Every new definition follows its structure exactly.

## File layout

```go
package s3  // = local type, underscores stripped for multi-word types

import (
    ".../destination/definitions"
    ".../destination/definitions/common"
    ".../destination/definitions/converter"
)

// 1. Source types: db-config supportedSourceTypes ∩ CLI ownership (the
//    event-stream types, plus warehouse whenever db-config lists it), as
//    common.SourceType* constants.
var sourceTypes = []string{
    common.SourceTypeAndroid,
    // ...
    common.SourceTypeCloud,
}

// 2. Connection modes: one entry per sourceTypes element (registry enforces
//    completeness). Values copied from db-config supportedConnectionModes.
var connectionModes = map[string][]string{
    common.SourceTypeAndroid: {"cloud"},
    // ...
}

// 3. Config struct: the closed allowlist of local YAML config keys.
//    mapstructure tag = snake_case local key; validate tag from schema.json;
//    default tag from schema.json `default` (see source-extraction.md
//    "Declaring defaults" — never on a `required` field).
type s3Config struct {
    BucketName    string `mapstructure:"bucket_name" validate:"required,min=1,max=100"`
    Prefix        string `mapstructure:"prefix" validate:"omitempty,max=100"`
    RoleBasedAuth *bool  `mapstructure:"role_based_auth" validate:"required"`
    // Conditionals: required_if / excluded_if reference the Go FIELD name.
    // Encode requiredness (required_if), not exclusion — presence is enforced
    // by validation, never by skipping empty values in the converter.
    IAMRoleARN  string `mapstructure:"iam_role_arn" validate:"required_if=RoleBasedAuth true,max=100"`
    AccessKeyID string `mapstructure:"access_key_id" validate:"required_if=RoleBasedAuth false,max=100"`
    AccessKey   string `mapstructure:"access_key" validate:"required_if=RoleBasedAuth false,max=100"`
    // schema.json defaults enableSSE to false. Optional + *bool, so it may
    // carry the tag; RoleBasedAuth above is `required` and must not.
    EnableSSE *bool `mapstructure:"enable_sse" default:"false"`
    // Mandatory type when consent is supported.
    ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
    // Present only when schema.json declares a connectionMode property, and
    // mandatory type when present (see source-extraction.md). db-config's
    // destConfig lists are not the signal — firebase names connectionMode
    // under all seven source types yet declares no schema property, so it
    // omits this field. Same integration pattern as consent — a common helper
    // plus a model field — but a different value shape (map[string]string,
    // not map[string][]ConsentEntry).
    // common.ConnectionModeProperties(sourceTypes) below does the mapping.
    // No validate tag: the legal values differ per source type, so
    // validateConnectionMode checks each entry against this definition's own
    // ConnectionModes map instead (see DEX-708 / ga4).
    ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// 4. NewDefinition: properties ported from terraform + consent + connection
// mode, then metadata.
func NewDefinition() *definitions.DestinationDefinition {
    properties := []converter.ConfigProperty{
        converter.Simple("bucketName", "bucket_name"),
        converter.Simple("prefix", "prefix"), // bare — no SkipZeroValue (see converter-mapping.md)
        // ...
    }
    properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
    properties = append(properties, common.Properties(sourceTypes)...)

    return &definitions.DestinationDefinition{
        Type:        "s3",        // lowercase(APIType), deterministic; equals the input localType
        DisplayName: "Amazon S3", // db-config.json "displayName"; registration rejects an empty one
        APIType:     "S3",        // upstream API type, derived from db-config.json "name" (Step 1)
        Version:    1,          // = terraform ConfigMeta.Version
        Properties: properties,
        SecretKeys: []string{"access_key_id", "access_key"}, // snake_case LOCAL keys
        NewConfig:  func() any { return &s3Config{} },
        SourceTypes:     append([]string(nil), sourceTypes...),
        ConnectionModes: connectionModes,
        // Per local source type and connection mode, the snake_case local
        // config keys required at connect time — derived from the
        // connectionMode-conditioned `allOf` branches in schema.json (see
        // source-type-mapping.md "Per-source-type connect-time required keys").
        // Omit the field when no source type contributes a key.
        ConnectionRequiredKeys: map[string]map[string][]string{
            common.SourceTypeWeb: {"cloud": {"api_key"}, "device": {"app_id"}},
        },
    }
}
```

## Source-type-gated config keys

When integrations-config `destConfig[<sourceType>]` exposes extra keys beyond
`defaultConfig` (e.g. Amplitude `eventUploadPeriodMillis` for android/ios/web,
Intercom `mobileApiKeyAndroid` for android), wrap the property in
`converter.Gated` — the local key is written once, in the constructor:

```go
converter.Gated(
    converter.Simple("eventUploadPeriodMillis", "event_upload_period_millis"),
    common.SourceTypeWeb, common.SourceTypeAndroid, common.SourceTypeIOS,
),
```

Do NOT gate the boilerplate consent/connection keys (`connection_mode`,
`consent_management`) — those stay handled by the existing source-type-keyed
block machinery. The registry builds a reverse index at
Register(); `RegisteredDefinition.GatedKeyPaths()` returns
`map[keypath][]sourceType` (JSON-pointer paths, e.g.
`/event_upload_period_millis`) for validation to consume.

## Rules the registry enforces at Register()

Violations fail `newDestinationRegistry` and thus every `cli/internal/app` test:

- `DisplayName` must not be empty.
- `NewConfig` must return a pointer to struct.
- Every `SourceTypes` entry must exist in the local→API source-type mapping.
- `ConnectionModes` keys ⊆ `SourceTypes`, and every source type must have modes.
- `ConnectionRequiredKeys` keys ⊆
  `SourceTypes`, every inner mode ∈ `ConnectionModes[sourceType]`, each key list
  non-empty, and every required key must exist on the config struct or be a
  source-type block key (`connection_mode`). Entries are
  optional per source type and per mode.
- `SyncBehaviours` entries ⊆ {`upsert`, `mirror`, `full`}, and any rETL
  metadata (`SyncBehaviours`, including an empty list, or
  `SupportsVisualMapper`) requires `warehouse` ∈ `SourceTypes`.
- A `consent_management` config field must be `common.ConsentManagement`, and a
  `connection_mode` field must be `common.ConnectionMode`. A bespoke type for
  either is rejected at registration: it would silently opt the key out of the
  shared validation those types drive.
- `(Type, Version)` and `(APIType, Version)` must be unique.
- Gated properties: source types ⊆ `SourceTypes`, local key must exist on the
  config struct, no duplicates, and the property must carry a local key
  (`Gated(Discriminator(...))` is rejected).
- Every `SecretKeys` path must resolve to a string field, through nested
  objects and lists of objects, and be snake_case, since each segment goes
  upper-cased into the secret's variable name.
- Every `validate` rule on the config struct must be one that
  `(*RegisteredDefinition).Describe` reads (`required`, `required_if`,
  `required_unless`, `required_with`, `excluded_with`, `excluded_if`, `oneof`,
  `dynamic_or_oneof`, `pattern`, `dynamic_or_pattern`, `min`, `max`, `dive`,
  `omitempty`) or a tag in the definition's `ConfigValidateFuncs`. Named
  patterns must be registered. This keeps `rudder-cli docs destination` able to
  describe every definition.
- Registration also rejects rule shapes the docs would misstate:
  - after `dive`, any rule but `oneof`, `dynamic_or_oneof`, `pattern`,
    `dynamic_or_pattern` or a custom tag;
  - `required` on a key inside a nested block that is not a pointer, since
    go-playground checks it even when the spec leaves the block out;
  - a `required*` rule on a `bool` or number that is not a pointer, since it
    rejects `false` or `0`;
  - a `required*` rule on a nested block that is not a pointer, since
    go-playground skips a bare `required` there and reads the other rules as
    "not empty";
  - a `required*` rule after `omitempty`, since go-playground skips every rule
    after `omitempty` on an empty value;
  - in a condition or a `oneof`, a bool value other than `true` or `false`
    (go-playground compares it with the text `true`) or an integer not written
    in plain base 10 (it reads `010` as octal);
  - a condition on a key that is not a string, bool or number;
  - `|`, a quoted value, or a `0x2C` or `0x7C` escape in a tag.

## Validation semantics to remember

- The struct is a closed allowlist: unknown local config keys error with
  `unknown config field`.
- `mapstructure` decode is strict (`WeaklyTypedInput: false`): YAML `"true"`
  string does not coerce to bool.
- `excluded_if=Field value` / `required_if=Field value` use the Go field name
  and the stringified value (`RoleBasedAuth true`).
- Real regex constraints use `validate:"pattern=<name>"` (default validator).
  Reuse an existing `NewPattern` or register a minimal one; strip upstream
  env/template alternations — never encode them into the CLI regex. See
  source-extraction.md "Enforcing regex patterns".
- Error paths are JSON pointers over local keys (`/bucket_name`,
  `/consent_management/web/0/provider`) — assert them in tests.

## Test template

`definitions/s3/definition_test.go` — three test functions:

1. `TestNewDefinitionMetadata` — register + `registry.Get(type, version)`,
   assert all metadata, `ConnectionModes` per source type, negative
   `NotContains` for dropped upstream source types, `GetByAPIType`.
2. `Test<Type>ConfigValidation` — parallel subtests through
   `registered.ValidateConfig(map[string]any{...})`, asserting `Path` and
   `Message` fragments.
3. `Test<Type>ConversionRoundTrip` — `testutil.AssertConversion(t,
   def.Properties, []testutil.ConversionCase{...})` with paired
   `LocalJSON`/`APIJSON`; include a consent case with boundary-mapped source
   types (`android_kotlin` ↔ `androidKotlin`, `react_native` ↔ `reactnative`).

## Registration in the builtin table

`newDestinationRegistry` registers the entries of the table in
`cli/internal/providers/destination/definitions/builtin/builtin.go`: the
verified ones always, the rest only under `UnverifiedDestinations`.

```go
var entries = []entry{
    {NewDefinition: s3.NewDefinition, Verified: true},
    // ...
    {NewDefinition: <pkg>.NewDefinition}, // the new definition, unverified
}
```

Note: `cli/internal/app/dependencies_test.go` asserts the exact
`registry.SupportedTypes()` slice — update that expectation when adding a
type (it is sorted alphabetically).

## Example YAML shape

`project-minimal/destinations/d1_s3.yaml` shows the spec envelope. The
generated example must use `definition_version` equal to the registered
`Version` and a config that passes `ValidateConfig`.
