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

// 1. Source types: db-config supportedSourceTypes ∩ CLI event-stream ownership,
//    as common.SourceType* constants.
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
//    mapstructure tag = snake_case local key; validate tag from schema.json.
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
    EnableSSE   *bool  `mapstructure:"enable_sse"`
    // Mandatory type when consent is supported.
    ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// 4. NewDefinition: properties ported from terraform + consent, then metadata.
func NewDefinition() *definitions.DestinationDefinition {
    properties := []converter.ConfigProperty{
        converter.Simple("bucketName", "bucket_name"),
        converter.Simple("prefix", "prefix"), // bare — no SkipZeroValue (see converter-mapping.md)
        // ...
    }
    properties = append(properties, common.Properties(sourceTypes)...)

    return &definitions.DestinationDefinition{
        Type:       "s3",       // lowercase(APIType), deterministic; equals the input localType
        APIType:    "S3",       // upstream API type, derived from db-config.json "name" (Step 1)
        Version:    1,          // = terraform ConfigMeta.Version
        Properties: properties,
        SecretKeys: []string{"access_key_id", "access_key"}, // snake_case LOCAL keys
        NewConfig:  func() any { return &s3Config{} },
        SourceTypes:     append([]string(nil), sourceTypes...),
        ConnectionModes: connectionModes,
        // Only when db-config has supportedSourcesValidation: per local source
        // type, the snake_case local config keys required at connect time.
        // Omit the field when upstream has none (the common case).
        SupportedSourcesValidation: map[string][]string{
            common.SourceTypeWeb: {"use_native_sdk"},
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
`use_native_sdk`, `consent_management`) — those stay handled by the existing
source-type-keyed block machinery. The registry builds a reverse index at
Register(); `RegisteredDefinition.GatedKeyPaths()` returns
`map[keypath][]sourceType` (JSON-pointer paths, e.g.
`/event_upload_period_millis`) for validation to consume.

## Rules the registry enforces at Register()

Violations fail `newDestinationRegistry` and thus every `cli/internal/app` test:

- `NewConfig` must return a pointer to struct.
- Every `SourceTypes` entry must exist in the local→API source-type mapping.
- `ConnectionModes` keys ⊆ `SourceTypes`, and every source type must have modes.
- `SupportedSourcesValidation` keys ⊆ `SourceTypes`, each entry non-empty, and
  every required key must exist on the config struct or be a source-type block
  key (`connection_mode`, `use_native_sdk`). Entries are optional per source
  type.
- A `consent_management` config field must be `common.ConsentManagement`.
- `(Type, Version)` and `(APIType, Version)` must be unique.
- Gated properties: source types ⊆ `SourceTypes`, local key must exist on the
  config struct, no duplicates, and the property must carry a local key
  (`Gated(Discriminator(...))` is rejected).

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

## Registration in dependencies.go

```go
func newDestinationRegistry(cfg config.Config) (*definitions.Registry, error) {
    registry := definitions.NewRegistry()
    if !cfg.ExperimentalFlags.DestinationSupport {
        return registry, nil
    }
    if err := registry.Register(s3.NewDefinition()); err != nil {
        return nil, fmt.Errorf("registering s3 destination definition: %w", err)
    }
    // Append the new definition here, same pattern.
    return registry, nil
}
```

Note: `cli/internal/app/dependencies_test.go` asserts the exact
`registry.SupportedTypes()` slice — update that expectation when adding a
type (it is sorted alphabetically).

## Example YAML shape

`project-minimal/destinations/d1_s3.yaml` shows the spec envelope. The
generated example must use `definition_version` equal to the registered
`Version` and a config that passes `ValidateConfig`.
