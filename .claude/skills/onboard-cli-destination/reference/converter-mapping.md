# Converter Mapping Cookbook

Porting terraform `configs.ConfigProperty` lists to the CLI
`definitions/converter` package. The CLI package was modeled on the terraform
one, so most calls port 1:1 — only the second argument's meaning changes from
"terraform key" to "local YAML key" (both snake_case, usually identical).

Package: `cli/internal/providers/destination/definitions/converter`

## Helper translation table

| Terraform (`rudderstack/configs`) | CLI (`definitions/converter`) | Notes |
| --- | --- | --- |
| `c.Simple(apiKey, tfKey, filters...)` | `converter.Simple(apiKey, localKey)` | Port **without** filters — see "Do not skip zero values" below. Terraform's `filters...` (`SkipZeroValue` etc.) are intentionally dropped for destination config |
| `c.SkipZeroValue` | **do not port** | Would omit zero values / empty slices from the API config and cause phantom, non-converging diffs. See "Do not skip zero values" below |
| `c.Conditional(apiKey, tfKey, cond)` | `converter.Conditional(apiKey, localKey, cond)` | Identical |
| `c.Equals(key, value)` | `converter.Equals(key, value)` | Identical |
| `c.Discriminator(apiKey, values)` | `converter.Discriminator(apiKey, values)` | Identical; `DiscriminatorValues` = `map[string]any` keyed by local key |
| `c.ArrayWithStrings(rootAPIKey, nestedField, tfKey)` | `converter.ArrayWithStrings(rootAPIKey, nestedField, localKey)` | Identical: `["a"]` ↔ `[{nestedField: "a"}]` |
| `c.ArrayWithObjects(rootAPIKey, tfKey, fields)` | `converter.ArrayWithObjects(rootAPIKey, localKey, fields)` | Identical; `fields` maps API field → local field name (string) or `converter.APINestedObject{LocalKey, NestedKey}` |
| `c.Negated(apiKey, tfKey)` | **no equivalent** | Rare. If needed, add `Negated` to the converter package (mirror the terraform implementation) rather than inlining a custom `ConfigProperty` in the definition |
| `GetCommonConfigMeta(sourceTypes)` | `common.Properties(sourceTypes)` | Consent management; CLI takes **local** source types |
| — (no terraform equivalent) | `converter.Gated(prop, sourceTypes...)` | Wrapper, not a mapping: restricts the property's local key to the given **local** source types. Use when db-config `destConfig` lists the API key only under specific source types (see source-extraction.md). Wraps any constructor except `Discriminator` (no local key — registry rejects it) |

## Do not skip zero values

**Port every `converter.Simple` bare — no `SkipZeroValue` or other value
filters**, even where terraform uses `c.SkipZeroValue`. Skipping empty values
from the API config produces phantom diffs that never converge. Enforce presence
through validation instead: `required` / `required_if` struct tags for fields
that must not be empty, `omitempty` / `*bool` for genuinely optional ones. The
merged S3 definition passes no filters to any `Simple`; match it.

## Dot-path API keys

gjson/sjson paths work the same in both: `"useNativeSDK.web"` writes/reads a
nested API object. Terraform's local side uses TF list indexing
(`use_native_sdk.0.web`); the CLI local side is plain YAML nesting, so use
`"use_native_sdk.web"`.

## Common patterns

Simple field:

```go
converter.Simple("prefix", "prefix")
```

Whitelist/blacklist with discriminator (GA4-style):

```go
converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering_whitelist"),
converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering_blacklist"),
converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
    "event_filtering_whitelist": "whitelistedEvents",
    "event_filtering_blacklist": "blacklistedEvents",
}),
```

List of objects with field rename:

```go
converter.ArrayWithObjects("piiPropertiesToIgnore", "pii_property", map[string]any{
    "piiProperty": "pii_property_name",
})
```

Same-shape list (no rename needed, e.g. webhook headers `{from, to}`):

```go
converter.Simple("headers", "headers")
```

## Config struct fields for each property shape

The converter maps values; the config struct validates them. Shapes must
agree with what the YAML holds locally:

| Local YAML shape | Struct field type |
| --- | --- |
| scalar string | `string` |
| optional bool | `*bool` |
| required bool (gates conditionals) | `*bool` + `validate:"required"` |
| string list (`ArrayWithStrings` local side) | `[]string` |
| object list | `[]struct{...}` with `mapstructure` tags per field, `validate:"omitempty,dive"` on the slice |
| consent block | `common.ConsentManagement` tagged `mapstructure:"consent_management"` (mandatory type) |

Unknown local keys are rejected by the validator automatically — the struct
is the closed allowlist, so every mapped property needs a struct field.
