# Source Type Mapping

Upstream `db-config.json` `supportedSourceTypes` values are the **API** source
types. The CLI definition uses **local** source types. The canonical mapping
lives in code — always read it, since it may gain entries:

`cli/internal/providers/destination/definitions/common/source_type_mapping.go`
(`apiSourceTypeByLocal`).

Current mapping (local → API):

| Local constant | Local value | API / db-config value |
| --- | --- | --- |
| `common.SourceTypeAMP` | `amp` | `amp` |
| `common.SourceTypeAndroid` | `android` | `android` |
| `common.SourceTypeAndroidKotlin` | `android_kotlin` | `androidKotlin` |
| `common.SourceTypeCloud` | `cloud` | `cloud` |
| `common.SourceTypeCloudSource` | `cloud_source` | `cloudSource` |
| `common.SourceTypeCordova` | `cordova` | `cordova` |
| `common.SourceTypeFlutter` | `flutter` | `flutter` |
| `common.SourceTypeIOS` | `ios` | `ios` |
| `common.SourceTypeIOSSwift` | `ios_swift` | `iosSwift` |
| `common.SourceTypeReactNative` | `react_native` | `reactnative` |
| `common.SourceTypeShopify` | `shopify` | `shopify` |
| `common.SourceTypeUnity` | `unity` | `unity` |
| `common.SourceTypeWarehouse` | `warehouse` | `warehouse` |
| `common.SourceTypeWeb` | `web` | `web` |

## Rules

- In `definition.go`, always reference the `common.SourceType*` constants,
  never string literals.
- Translate each upstream `supportedSourceTypes` entry to its local constant.
  Upstream values with no row here (e.g. `tiktokAds`, `singer-*`): drop them
  and flag in the final report — `registry.Register` fails on any local source
  type missing from the mapping, so guessing breaks the build anyway.
- The CLI intentionally supports a subset. Never declare `amp`, `shopify`,
  `warehouse` or `cloud_source`, even when upstream lists them — the CLI cannot
  produce those tokens, so a destination declaring one can never have a matching
  connection validated. `SourceSpec`
  (`cli/internal/providers/event-stream/source/model.go`) carries no category
  field and constrains `type` to the SDK definitions, and the sole
  `common.SourceTypeToken` call site
  (`event-stream/rules/connection/connection_semantic_valid.go`) passes an empty
  category, leaving the `SourceCategoryCloud`/`Singer` → `cloud_source` and
  `SourceCategoryWarehouse` → `warehouse` branches dead. What remains is the ten
  types S3 declares: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`,
  `unity`, `cloud`, `react_native`, `flutter`, `cordova`. `customerio_audience`
  is the sole exception, since `warehouse` is its only source type (DEX-720).
  When unsure about another mapped-but-unusual type, include what S3 includes
  and flag the rest.
- `ConnectionModes` must be keyed by the same local types and cover every
  entry in `SourceTypes` — registration errors otherwise. Copy the modes per
  source type from db-config `supportedConnectionModes` (values are `cloud`,
  `device`, `hybrid`).
- `SupportedSourcesValidation` is derived from the **per-source-type key lists
  inside `config.destConfig`** — see "Per-source-type config keys" below.
- Consent management uses the same mapping automatically via
  `common.Properties(sourceTypes)` — no extra work per source type.
- Connection mode uses the same mapping automatically via
  `common.ConnectionModeProperties(sourceTypes)` — no extra work per source
  type. Its values are validated against this destination's own
  `ConnectionModes` map, not a fixed enum.

## Per-source-type connect-time required keys

`SupportedSourcesValidation` is
`map[localSourceType]map[connectionMode][]localConfigKey` — the config keys a
destination must carry for a source of that type, connected in that mode, to be
valid. Both dimensions matter: Braze needs `rest_api_key` from a cloud-mode
source and `app_key` from a device-mode one, so source type alone cannot decide.

**The source is `schema.json`, not `db-config.json`.** There is no
`supportedSourcesValidation` key in db-config — the name belongs to the
config-backend destination-definition entity (`connection.service.ts` reads
`destDefConfig.supportedSourcesValidation`) and no upstream definition
populates it (zero occurrences across rudder-integrations-config, verified
2026-08-25). Never search db-config for it. `config.destConfig.<sourceType>`
lists which keys are *scoped* to a source type; it does not say which are
*required*, so it is not the source either — it feeds gated-key detection only
(source-extraction.md "Detecting gated keys in destConfig").

### Where the conditions live

`configSchema.allOf` is a list of `{if, then}` branches. `then.required` is the
key list; `if` says when it applies. Only branches conditioned on
`connectionMode` belong here.

- **`configSchema.required` (top level)** — unconditional, already enforced by
  `validate:"required"` struct tags. Never copy into the map.
- **A `then` with `properties` but no `required`** — makes a key *available*,
  not required (Braze's `usePlatformSpecificApiKeys` branch). Skip it.
- **An `if` on ordinary config keys** (GA4 `typesOfClient`, Customer.io
  `apiVersion`) — ordinary conditional requiredness; it becomes a
  `required_if` / `required_unless` tag, never a map entry. See
  source-extraction.md "Conditional requiredness".

### Recognised `if` shapes

| Shape | Example | Read as |
| --- | --- | --- |
| `if.properties.connectionMode.anyOf[]`, each branch `{properties: {<apiSourceType>: {const: <mode>}}, required: [<apiSourceType>]}` | Braze | the union of the listed `(source type, mode)` pairs |
| `if.properties.connectionMode.properties`, mapping `<apiSourceType>` → `{const: <mode>}` (usually with `additionalProperties: false`) | Intercom | each `(source type, mode)` pair in the object |
| `if.not { … connectionMode … }` | Facebook Pixel: `not(connectionMode.web == "device")` → `accessToken` | every supported `(source type, mode)` pair **except** the ones the negated clause matches |
| `if.properties` carries `connectionMode` **and** another config key (Braze `usePlatformSpecificApiKeys: {const: true}`, `if.required: ["usePlatformSpecificApiKeys", "connectionMode"]`) | Braze `appKey` / `androidApiKey` / `iOSApiKey` / `webApiKey` | **not expressible as a `SupportedSourcesValidation` entry** — the map has no room for a value-dependent condition. Express the whole branch as a custom validator instead (see "Expressing it as a custom validator instead" below) and drop it from this map |

### Expressing it as a custom validator instead

The last row above is unexpressible only *as a `SupportedSourcesValidation`
entry* — that map has one key per `(source type, mode)` pair and no room for a
condition on another config key. Since every destination models
`connection_mode` as real, validated config (source-extraction.md's
`connectionMode` note; DEX-708's `ga4` pilot established the pattern), the
branch is expressed directly as ordinary config validation instead: a custom
go-playground tag, scoped to that one definition via
`DestinationDefinition.ConfigValidateFuncs`, whose `validator.Func` reads both
the sibling config key and `connection_mode` off `FieldLevel.Parent()` via
reflection — the technique `ga4`'s `sdkBaseURLConditional` uses for its
`sdk_base_url` pattern condition, adapted to return presence (`value != ""`)
instead of a pattern match. See source-extraction.md "Conditional
requiredness" for the full writeup and the pointer-field caveat.

### Derivation

1. Walk `configSchema.allOf`. For each branch with a non-empty `then.required`
   and a `connectionMode`-based `if`, expand the `if` into `(apiSourceType,
   mode)` pairs per the table.
2. Translate source types to local via the table above; drop pairs whose source
   type is not in `SourceTypes`.
3. Drop pairs whose mode is not listed for that source type in
   `ConnectionModes` (db-config `supportedConnectionModes`).
4. Translate each `then.required` API key to its snake_case local key. Every key
   must be a `mapstructure` tag on the config struct — one that is not means the
   property was dropped or renamed; re-check before excluding it.
5. Union the key lists when several branches hit the same `(source type, mode)`;
   dedupe and keep each list sorted.
6. Omit a mode with no keys, a source type with no modes, and the whole field
   when nothing survives.

### Worked example — Intercom

Two branches: `connectionMode` device for `android`/`ios`/`web` → `appId`;
`connectionMode` cloud for `amp`/`android`/`cordova`/`flutter`/`ios`/
`reactnative`/`shopify`/`unity`/`web` → `apiKey`. Dropping `amp` and `shopify`
(not in `SourceTypes`):

```go
SupportedSourcesValidation: map[string]map[string][]string{
    common.SourceTypeAndroid:     {"cloud": {"api_key"}, "device": {"app_id"}},
    common.SourceTypeIOS:         {"cloud": {"api_key"}, "device": {"app_id"}},
    common.SourceTypeWeb:         {"cloud": {"api_key"}, "device": {"app_id"}},
    common.SourceTypeUnity:       {"cloud": {"api_key"}},
    common.SourceTypeReactNative: {"cloud": {"api_key"}},
    common.SourceTypeFlutter:     {"cloud": {"api_key"}},
    common.SourceTypeCordova:     {"cloud": {"api_key"}},
}
```

Flag in the report: the cloud branch omits `androidKotlin`, `iosSwift` and
`cloud`, which the CLI supports in cloud mode — an upstream gap, so those source
types get no entry rather than a guessed `api_key`.

### Constraints

Keys ⊆ `SourceTypes`; every inner mode ∈ `ConnectionModes[sourceType]`; every
key list non-empty; every key a `mapstructure` tag on the config struct. Drop
entries whose source type was dropped from `SourceTypes` and flag them.

Evaluating the map at connect time needs the destination spec to declare
`connection_mode.<source type>` for the connecting source; a destination that
supports device or hybrid mode and omits it cannot be resolved — flag that too.
