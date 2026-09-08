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
| `minLength`/`maxLength` keywords (not a pattern) | `min=N` / `max=N` |
| `enum: ["a","b"]` | `oneof=a b`, unless schema.json declares a `{{ … \|\| … }}` alternative for that field — then `dynamic_or_oneof=a b`. See "Let schema.json decide whether an enum may be dynamic" below |
| any `pattern` | `dynamic_or_pattern=<name>` via the shared named-pattern registry (see "Enforcing regex patterns" below): passes `{{ path \|\| fallback }}` templates, otherwise enforces the named pattern. Use plain `pattern=<name>` only when the field must reject templates too |
| `minLength`/`maxLength` | `min=N` / `max=N` |
| `if/then` or `allOf` conditionals | `required_if=Field value`, `excluded_if=Field value` (see S3 `iam_role_arn`/`access_key` for the pattern). See "Conditional requiredness" below — the built-ins cover every shape upstream uses |

Notes:

- Strip the template/env prefix from upstream patterns before deciding what to
  enforce. Upstream almost always wraps the real constraint as
  `(^\{\{.*\|\|(.*)\}\}$)|(^env[.].+)|<real>`. **The named pattern must contain
  only `<real>`** — never re-encode either branch into the regex. The template
  branch is handled by the tag: `dynamic_or_pattern=<name>` accepts
  `{{ path || fallback }}` and otherwise enforces `<real>`. Example: schema
  `(^\{\{.*\|\|(.*)\}\}$)|(^env[.].+)|^AW-(.{0,100})$` → register / reuse
  `^AW-(.{0,100})$` and tag `validate:"required,dynamic_or_pattern=<name>"`.
- **`<real>` is a regex even when it looks like a length limit.** `^(.{0,100})$`
  bounds length *and* forbids line breaks, so `max=100` is not equivalent — it
  lets newlines through. Register the pattern (the shared `single_line_100`
  already covers this very common shape) instead of translating to `max=N`. Only
  a genuine `minLength`/`maxLength` keyword becomes `min`/`max`.
- Routing a field through `dynamic_or_pattern` also lifts the length cap for
  templates, matching upstream: the template branch carries no cap, so
  `{{ config.x || … }}` longer than the literal limit stays valid.
- **`env.VAR` is deprecated — never give it an escape hatch.** It is judged as an
  ordinary literal, so it passes only when it happens to satisfy the pattern.
  Do not add it to a pattern or a tag. It also plays no part in deciding whether
  a property admits a dynamic value: upstream writes these wrappers as
  `(^\{\{.*\|\|(.*)\}\}$)|(^env[.].+)|<real>`, and only the `{{ … || … }}` branch
  counts. Disregard `(^env[.].+)` wherever it appears — in schema.json and in
  terraform validators alike.
- **Let schema.json decide whether an enum may be dynamic.** Pick the tag from
  what the property declares, not from the shape of the constraint: if it admits
  the `{{ path || fallback }}` form, use `dynamic_or_oneof`; if it does not, use
  `oneof`, since such a value would then be stored verbatim and rejected by the
  backend. In practice this currently splits by shape — `pattern` properties
  carry the template branch, `enum` properties do not (checked across every
  onboarded destination), so enums take `oneof` today — but check the property
  rather than assuming, since an enum that did declare it would take
  `dynamic_or_oneof`. terraform shows the same split: its enum validators carry
  no `{{ … || … }}` branch where its pattern validators do. Many existing
  definitions use `dynamic_or_oneof` for strict enums; reconciling them is tracked
  separately, so check the schema rather than copying a neighbouring definition.
- Booleans that gate conditionals (like S3 `role_based_auth`) should be
  `*bool` with `validate:"required"` so "absent" and "false" are distinct.
- Optional booleans → `*bool` without `required`.
- `consentManagement` and `connectionMode` subtrees in schema.json are handled
  by `common` — do not model either as an ad-hoc field. Append
  `common.Properties(sourceTypes)` and add a `ConsentManagement` field when the
  destination supports consent; append `common.ConnectionModeProperties(sourceTypes)`
  and add a `ConnectionMode` field **only when schema.json declares a
  `connectionMode` property** (see definition-anatomy.md). Where it does,
  `connection_mode` persists as real, validated destination config — DEX-708's
  `ga4` pilot established the pattern, since rolled out to every
  schema-declaring destination; see "Conditional requiredness" below for the
  connectionMode-and-config-key conditional this unlocks. Where it does not,
  omit it: db-config's `destConfig.<sourceType>` lists name `connectionMode` for
  far more destinations than schema.json constrains it for, so the destConfig
  lists are not the signal. `firebase` is the worked example — every one of its
  seven source types lists `connectionMode` in db-config, and schema.json
  declares no such property, so it stays unmodelled. Neither key is ever wrapped
  in `Gated` — both stay handled by the source-type-keyed block machinery, not
  the `Gated`-scan below.
- A property marked `"rs-immutable": true` is still modelled and validated
  normally — immutability constrains *updates*, not the config surface. Record
  which keys carry it: the backend 400s on any change to one, and the e2e update
  fixture must leave them untouched (see e2e-tests.md).

## Declaring defaults

The control plane applies `schema.json` `default` values when it persists a
destination (its validator runs with `useDefaults: true`, mutating the config it
validates). A defaulted key is therefore **never optional in storage** — the
backend fills it in. If the CLI does not declare the same default, a spec that
omits the key diffs forever against the remote state that carries it, and an
update drops the key upstream.

Declare each default as a `default:"…"` struct tag beside the field's
`mapstructure`/`validate` tags. The registry parses them once at `Register()`
and `ApplyDefaults` fills only the keys a spec omitted, so an explicit value —
including one equal to the zero value — always wins.

### Finding every default

Do **not** read `configSchema.properties` alone. Descend:

- **Combinators** — `allOf`, `anyOf`, `oneOf`, `if`, `then`, `else`,
  `dependentSchemas` — describe the *same* config object, so a default found
  inside one still belongs to a top-level key. Several destinations declare
  defaults only here (`SNOWFLAKE`, `RS`, `GOOGLEADS`, `KAFKA`, `POSTGRES`,
  `REDIS`, `HS`).
- `properties.<name>` and `items` describe the level *below* — a default there
  is nested, not top level.

### Writing the tag

The schema's `"type"` picks the Go field type; the tag value is always a string:

| schema type | Go field | tag |
| --- | --- | --- |
| `"boolean"` | `*bool` (pointer keeps absent ≠ false) | `default:"false"` |
| `"string"` | `string` | `default:"cloud"` |
| `"integer"` | `int` | `default:"2"` |

Only these three are implemented — they are the only types `default` takes
across the destinations we model. A `uint`, float, slice or map field is
rejected at registration rather than guessed at; extend `parseDefaultValue` when
a schema first needs one.

**Let the schema type decide, not how the value looks.** `syncFrequency` is
`"type": "string"` with default `"180"` in `BQ`, `POSTGRES`, `RS`,
`S3_DATALAKE` and `SNOWFLAKE` — declare it `string`. An `int` field would store
`180` where the API returns `"180"`, producing a permanent diff.

### When NOT to declare one

- **The field is `validate:"required"`.** Registration rejects the combination:
  a key that must always be present can never take a default. This includes
  fields the CLI marks required even when `schema.json` does not (e.g.
  `s3_datalake` `use_glue` / `role_based_auth`, deliberately `*bool` +
  `required` so absent stays distinct from false). `required_if` is fine — that
  field is genuinely optional.
- **The key has no local config key.** `eventFilteringOption` is defaulted by 21
  destinations but is derived by `converter.Discriminator` from the
  whitelist/blacklist arrays, so it cannot carry a tag.
  terraform-provider-rudderstack neither sends nor reads it; match that.
- **The default is nested or inside array `items`.** `ApplyDefaults` is
  top-level only. Flag these in the final report instead.

There is no shared catalogue to copy from — scan the destination's own
`schema.json` with the traversal above, and record in the final report every
default you found but did not declare, with the reason.

## Conditional requiredness

**First, route the branch.** An `allOf` branch whose `if` tests
`connectionMode` states a requirement that depends on the *connected source*,
not on the config alone — no struct tag can express it. Those branches become
`ConnectionRequiredKeys` entries instead (source-type-mapping.md
"Per-source-type connect-time required keys"), **except** a branch whose `if`
also carries another config key (e.g. Braze's `usePlatformSpecificApiKeys`):
that shape has no room in `ConnectionRequiredKeys`'s map, so express it
directly as a custom validator instead — see "The one exception" below and
source-type-mapping.md "Expressing it as a custom validator instead".
Everything else in this section is for branches conditioned on ordinary
config keys.

`schema.json` states conditional requiredness as `allOf` branches, and
go-playground's built-in tags cover every shape upstream uses. **Never write a
custom validator or a `CustomValidateConfig`-style hook for this** — see the
worked example in `definitions/postgres/definition.go`, which enforces all eight
of its branches with built-ins alone.

**The one exception:** `required_if`/`excluded_if` resolve conditions against
direct struct field names only, so a condition keyed on a **map field** — in
practice this means `connection_mode.<sourceType>`, where the destination models
it per the note above — cannot be expressed with a built-in tag, whether the
thing being gated is a pattern (`ga4`'s `sdk_base_url`, conditioned on
`client_type` **and** `connection_mode.web`) or plain requiredness (Braze's
`app_key` / `android_api_key` / `ios_api_key` / `web_api_key`, conditioned on
`use_platform_specific_api_keys` **and** `connection_mode.<sourceType>` — see
source-type-mapping.md "Recognised `if` shapes").

There, register a custom tag scoped to the one definition via
`DestinationDefinition.ConfigValidateFuncs` in `NewDefinition` (never the global
`vrules.RegisterDefaultValidator` — that registry is shared by every
destination's validation call and reserved for fleet-wide conventions like
`pattern`/`dynamic_or_pattern`, so a one-destination condition doesn't belong
there), whose `validator.Func` reads the sibling fields off `FieldLevel.Parent()`
by name. `ga4`'s `sdkBaseURLConditional` (DEX-708) is the worked example for a
pattern condition; a requiredness version follows the identical shape but
returns whether the target field is non-empty once the condition holds, instead
of matching a pattern. This does **not** lift the pointer restriction below: the
target field must still be a plain type (`string`, not `*bool`), since
go-playground never invokes a custom tag's function for a nil pointer. The error
message falls through to `funcs/utils.go`'s generic `default` case (the raw
go-playground message); that is acceptable for a rare, narrowly-scoped tag and
isn't worth widening the shared formatter for.

Where `connection_mode` is modelled, it changes what a CLI apply sends for that
key: an existing destination set up via the UI needs its spec to declare
`connection_mode` before its first CLI-managed apply, or that apply drops it
(update replaces the whole config object).

Three facts settle almost every branch:

1. **`required_if` ANDs multiple field/value pairs.** A branch guarded by two
   conditions is one tag, not a special case:
   `if {useRudderStorage: false, bucketProvider: MINIO} -> require endPoint`
   becomes `validate:"required_if=UseRudderStorage false BucketProvider MINIO"`.
2. **`required_unless` covers "required for a subset of values".** `required_if`
   ANDs its pairs and panics on a repeated field, so it cannot say "required for
   S3, GCS or MINIO". State the rule as its inverse instead — `required_unless`
   exempts when **any** pair matches:
   `validate:"required_unless=UseRudderStorage true BucketProvider AZURE_BLOB"`.
   Verify the inversion against the branch you are translating; write a test that
   pins both the exemption and a provider that still requires the key, because
   nothing else distinguishes the two tags.
3. **Always include the guard condition.** Every per-provider branch is also
   gated on the storage/auth selector (`useRudderStorage: false`). Omitting it
   lets a stale selector — cleared config keys persist upstream — resurrect
   requirements on a config that no longer uses that provider, which rejects
   something the API accepts.

`required_unless` reports the key on a config whose selector is absent or
invalid, since that falls outside every exemption. That is extra detail on a
config already rejected for its selector, never a false rejection — pin it with a
test so it reads as intentional.

**Pointer fields must use built-in tags.** validator dereferences a non-nil
pointer before invoking a *custom* tag's function, and fails a nil pointer
carrying one before the function runs at all. A `*bool` under a custom tag
therefore reports an explicitly-set `false` as missing and an absent value as
missing regardless of the condition. The built-ins are unaffected. This matters
because gating booleans are `*bool` by convention (see the note above), so
`required_if` / `required_unless` are the only correct choice for them.

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
3. Only if the constraint is a genuine `minLength`/`maxLength` keyword or an
   enum fall back to `min`/`max` / `oneof` / report note.

### Tag usage

```go
// required shape, templates allowed (the default)
ConversionID string `mapstructure:"conversion_id" validate:"required,dynamic_or_pattern=googleads_conversion_id"`

// optional shape, templates allowed
ServerURL string `mapstructure:"server_url" validate:"omitempty,dynamic_or_pattern=url"`

// templates must be rejected too — justify this in the report
StrictID string `mapstructure:"strict_id" validate:"omitempty,pattern=some_strict_id"`
```

`dynamic_or_pattern` is a strict superset of `pattern`: a nil optional pointer
passes, a `{{ path || fallback }}` template passes, and everything else — empty
strings and `env.VAR` included — is decided by the named pattern. Both tags share
one registry, so a name registered with `funcs.NewPattern` /
`NewPatternWithReject` works with either, reject patterns included.

### Reject patterns are not allow patterns

`NewPatternWithReject` takes a second regex that *blocks* a value (the RE2
translation of an upstream negative lookahead, e.g. `(?!.*.ngrok.io)`). The
anchoring rules invert for it:

- **Do not anchor a reject to the ends.** An allow pattern is safer anchored; a
  reject is safer matching broadly, because narrowing it creates bypasses.
  Anchoring an ngrok block as `^(.*\.)?ngrok\.io$` lets `host.ngrok.io.` (a valid
  trailing-dot FQDN) and `host.ngrok.io:5432` straight through. Write it as
  `^.*<constraint>.*$` — explicit for readers and for CodeQL's
  `go/regex/missing-regexp-anchor`, identical in behaviour to leaving it bare.
- **Reproduce upstream's regex exactly, escaping included.** Upstream lookaheads
  are frequently written with unescaped dots (`.ngrok.io` matches any character
  where you might expect a literal `.`). "Fixing" that to `\.ngrok\.io` narrows
  the block and lets through hosts the API rejects, which defers the failure to
  apply — the opposite of what this validation is for. Compare destinations
  before assuming a form is canonical: postgres leaves its dots unescaped while
  redis and slack escape theirs.
- Confirm parity rather than eyeballing it: run the upstream regex and the CLI
  pattern over the same host list and check every verdict matches.

### Tests

For every pattern-validated field: add `ValidateConfig` subtests that a clearly
invalid literal is rejected (path + message fragment) and that a
`{{ path || fallback }}` template is accepted (or rejected, for plain
`pattern=`). Keep the valid example YAML using a literal that satisfies the
pattern.

For a reject pattern, also cover the shapes that an over-anchored version would
miss (trailing dot, `host:port` suffix) and a value that merely resembles the
blocked one but must pass.

## 3. db-config.json — capabilities

File: `rudder-integrations-config/src/configurations/destinations/<dir>/db-config.json`

| Extract | From | Use as |
| --- | --- | --- |
| Secrets | `config.secretKeys` (camelCase) | `SecretKeys` — **every entry**, translated to snake_case local keys. db-config is authoritative for which values are write-only; terraform's `Sensitive: true` is only a cross-check and is often incomplete |
| Source types | `config.supportedSourceTypes` | `SourceTypes` — translate via source-type-mapping.md; drop unmapped, flag in report |
| Connection modes | `config.supportedConnectionModes` (map per source type) | `ConnectionModes` keyed by **local** source type. Every entry in `SourceTypes` must have modes — the registry rejects gaps |
| Field allowlist | `config.destConfig.defaultConfig` | Sanity check: every mapped property's API key should appear here OR in a per-source-type list (then it is gated, see below); in neither → flag |
| Source-type-gated keys | `config.destConfig.<sourceType>` lists | API keys that appear only under specific source types (not in `defaultConfig`) are gated: wrap the ported property in `converter.Gated(prop, localSourceTypes...)` |

db-config has **no** `supportedSourcesValidation` key — that name exists only on
the config-backend entity, and `destConfig` says which keys are *scoped* to a
source type, never which are *required*. Connect-time required keys come from
`schema.json`; see source-type-mapping.md "Per-source-type connect-time required
keys".

**A secret you cannot express is a signal the local shape is wrong, not a
licence to drop it.** `SecretKeys` holds local key paths, so every entry in
`config.secretKeys` must correspond to a modelled local key. If one does not,
re-check the shape before excluding it — the usual cause is a local model that
nests keys the upstream config keeps flat. Snowflake is the worked example: its
eight secrets looked unreachable under terraform-shaped `s3`/`gcp`/`azure`
blocks, and all eight became ordinary top-level keys once the model matched the
flat upstream payload. Genuinely nested secrets (an array element such as
`headers.to`) are addressable by dotted path; excluding a declared secret is a
last resort that must be flagged prominently in the report.

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

Skip the boilerplate keys when doing this scan — none of them is ever wrapped
in `Gated`, though for two different reasons. `connectionMode`, `useNativeSDK`,
`consentManagement`, `eventFilteringOption`, `whitelistedEvents` and
`blacklistedEvents` are handled by the `common` package, the converter and the
source-type-keyed block machinery. `oneTrustCookieCategories` and
`ketchConsentPurposes` are not handled at all — they are deliberately left
unmodelled because the backend rewrites them into `consentManagement` on write
(see the migration-on-write exception in SKILL.md).

Example (Intercom): `mobileApiKeyAndroid` appears only in
`destConfig.android` → `converter.Gated(converter.Simple("mobileApiKeyAndroid", "mobile_api_key_android"), common.SourceTypeAndroid)`.

## Cross-checks (flag, never silently resolve)

- Terraform property with no schema.json entry, or vice versa.
- `Sensitive: true` in terraform but key absent from `secretKeys` (or reverse).
- Terraform `supportedSourceTypes` differing from db-config's list — db-config
  wins for capabilities, but note the difference.
- Upstream fields present in db-config `defaultConfig` but never mapped in
  terraform: omit from the CLI definition and list in the report.
