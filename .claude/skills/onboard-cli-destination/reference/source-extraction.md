# Source Extraction

What to pull from each upstream source, per its fixed role.

## 1. Terraform provider — property mappings

File: `terraform-provider-rudderstack/rudderstack/integrations/destinations/destination_<name>.go`

Extract from the `init()` function:

| Extract | From | Use as |
| --- | --- | --- |
| `APIType` | `c.Destinations.Register(name, c.ConfigMeta{APIType: ...})` | `APIType` (must equal the value derived from db-config.json in Step 1) |
| `Version` | `ConfigMeta.Version` | `Version` |
| `[]c.ConfigProperty` list | `properties := ...` | Port 1:1 to CLI converter (see converter-mapping.md) |
| `Sensitive: true` schema fields | `map[string]*schema.Schema` | Cross-check against db-config `secretKeys` |
| Regex in `c.StringMatchesRegexp(...)` | schema validators | Cross-check against schema.json patterns |

Skip anything produced by `GetCommonConfigMeta(...)` (consent management) —
the CLI equivalent is `common.Properties(sourceTypes)`.

Terraform keys are already snake_case; local CLI keys are the same
snake_case keys. API keys are the camelCase first arguments.

## 2. schema.json — validations

File: `rudder-integrations-config/src/configurations/destinations/<dir>/schema.json`
(root object is `configSchema`, JSON Schema draft-07).

Translate constraints to `validate` struct tags on the config struct:

| schema.json | validate tag |
| --- | --- |
| key in `required` | `required` |
| optional string with max | `omitempty,max=N` (schema patterns like `^(.{0,100})$` → `max=100`) |
| `enum: ["a","b"]` | `dynamic_or_oneof=a b` (custom tag in `definitions/dynamicvalues.go`: passes `env.VAR` / `{{ ... }}` dynamic values, otherwise enforces the enum). Use plain `oneof=a b` only when the field can never hold a dynamic value |
| `pattern` (real regex, not just length) | `pattern=<name>` via the shared named-pattern registry (see "Enforcing regex patterns" below). Length-only patterns stay as `max=N` / `min=N` — do not register a regex for those |
| `minLength`/`maxLength` | `min=N` / `max=N` |
| `if/then` or `allOf` conditionals | `required_if=Field value`, `excluded_if=Field value` (see S3 `iam_role_arn`/`access_key` for the pattern) |

Notes:

- Strip the template/env prefix from upstream patterns before deciding what to
  enforce. Upstream often wraps the real constraint as
  `(^\{\{.*\|\|(.*)\}\}$)|(^env[.].+)|<real>` so terraform accepts UI/env
  references. **CLI destination config must enforce only `<real>`** — never
  re-encode the `env.` / `{{ ... }}` branches into the named pattern, and never
  invent a `dynamic_or_pattern` (or similar) escape hatch for regex fields.
  Example: schema
  `(^\{\{.*\|\|(.*)\}\}$)|(^env[.].+)|^AW-(.{0,100})$` → register / reuse
  `^AW-(.{0,100})$` and tag `validate:"required,pattern=<name>"`.
- Booleans that gate conditionals (like S3 `role_based_auth`) should be
  `*bool` with `validate:"required"` so "absent" and "false" are distinct.
- Optional booleans → `*bool` without `required`.
- `consentManagement` and `connectionMode` subtrees in schema.json are
  handled by `common` — do not model them as ad-hoc fields.

## Enforcing regex patterns

Destination config validation already supports `validate:"pattern=<name>"`
because `GetPatternValidator()` is registered as a default validator in
`cli/internal/provider/rules/funcs/init.go`. Use named patterns — do not
inline raw regex in struct tags, and do not leave real patterns "unenforced"
in the report when a minimal named pattern can express them.

### Decision ladder (use the first that fits)

1. **Reuse an existing named pattern.** Search `NewPattern(` under
   `cli/internal/provider/rules/funcs/` and
   `cli/internal/providers/destination/` (and any sibling definition
   packages). Prefer shared names already in use (`url`,
   `destination_display_name`, etc.) when the constraint matches.
2. **Register a new minimal named pattern** when no existing name fits.
   - Shared / cross-destination constraints →
     `cli/internal/provider/rules/funcs/init.go` (same place as `url`).
   - Destination-specific constraints → `init()` (or a tiny `patterns.go`)
     next to that destination's `definition.go`, calling `funcs.NewPattern`.
   - Name: short, snake_case, destination-scoped when specific
     (e.g. `googleads_conversion_id`).
   - Regex: **only** the real value constraint after stripping upstream
     env/template alternations. Keep it minimal — no `(env.…)|` /
     `(\{\{…\}\})|` branches.
   - Error message: short, user-facing (what the value must look like).
3. Only if the constraint cannot be expressed as a regex (or is length-only
   / enum-only) fall back to `min`/`max` / `dynamic_or_oneof` / report note.

### Tag usage

```go
// required literal shape
ConversionID string `mapstructure:"conversion_id" validate:"required,pattern=googleads_conversion_id"`

// optional literal shape
ServerURL string `mapstructure:"server_url" validate:"omitempty,pattern=url"`
```

### Tests

For every `pattern=` field: add a `ValidateConfig` subtest that a clearly
invalid literal is rejected (path + message fragment), and keep the valid
example YAML using a literal that satisfies the pattern.

## 3. db-config.json — capabilities

File: `rudder-integrations-config/src/configurations/destinations/<dir>/db-config.json`

| Extract | From | Use as |
| --- | --- | --- |
| Secrets | `config.secretKeys` (camelCase) | `SecretKeys` — translate to snake_case local keys. **Flat top-level keys only**: the CLI secret machinery (`wrapKnownSecrets`/`maskSecrets` in `handler.go`) replaces top-level string values. Nested paths like `headers.to` cannot be modeled — leave them out of `SecretKeys` and flag in the report |
| Source types | `config.supportedSourceTypes` | `SourceTypes` — translate via source-type-mapping.md; drop unmapped, flag in report |
| Connection modes | `config.supportedConnectionModes` (map per source type) | `ConnectionModes` keyed by **local** source type. Every entry in `SourceTypes` must have modes — the registry rejects gaps |
| Field allowlist | `config.destConfig.defaultConfig` | Sanity check: every mapped property's API key should appear here OR in a per-source-type list (then it is gated, see below); in neither → flag |
| Source-type-gated keys | `config.destConfig.<sourceType>` lists | API keys that appear only under specific source types (not in `defaultConfig`) are gated: wrap the ported property in `converter.Gated(prop, localSourceTypes...)` |

## Detecting gated keys in destConfig

`destConfig` merges `defaultConfig` + `destConfig[<connected sourceType>]`.
For each terraform-mapped property's API key:

1. In `defaultConfig` → default key, port normally.
2. Not in `defaultConfig` but in one or more `destConfig.<sourceType>` lists →
   gated. Collect the source types listing it, translate to local source types
   via source-type-mapping.md, and wrap: `converter.Gated(prop, localTypes...)`.
   If a listing source type is dropped (unmapped/not CLI-owned), exclude it
   from the gate; if ALL its source types are dropped, omit the property and
   flag in the report.
3. In no list at all → flag as discrepancy.

Skip the boilerplate keys when doing this scan — `connectionMode`,
`useNativeSDK`, `consentManagement`, `oneTrustCookieCategories`,
`ketchConsentPurposes`, `eventFilteringOption`, `whitelistedEvents`,
`blacklistedEvents` are handled by the `common` package and the
source-type-keyed block machinery, never by `Gated`.

Example (Intercom): `mobileApiKeyAndroid` appears only in
`destConfig.android` → `converter.Gated(converter.Simple("mobileApiKeyAndroid", "mobile_api_key_android"), common.SourceTypeAndroid)`.

## Cross-checks (flag, never silently resolve)

- Terraform property with no schema.json entry, or vice versa.
- `Sensitive: true` in terraform but key absent from `secretKeys` (or reverse).
- Terraform `supportedSourceTypes` differing from db-config's list — db-config
  wins for capabilities, but note the difference.
- Upstream fields present in db-config `defaultConfig` but never mapped in
  terraform: omit from the CLI definition and list in the report.
