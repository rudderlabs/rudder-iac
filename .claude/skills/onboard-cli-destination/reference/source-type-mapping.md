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
- `SupportedSourcesValidation` (from db-config `supportedSourcesValidation`,
  when present) must be keyed by the same local types, each a member of
  `SourceTypes`, with non-empty snake_case local config keys that exist on the
  config struct or are source-type block keys (`connection_mode`,
  `use_native_sdk`) — registration errors otherwise. Entries are optional per
  source type; drop entries whose source type was dropped and flag them.
- Consent management uses the same mapping automatically via
  `common.Properties(sourceTypes)` — no extra work per source type.
- Connection mode uses the same mapping automatically via
  `common.ConnectionModeProperties(sourceTypes)` — no extra work per source
  type. Its values are validated against this destination's own
  `ConnectionModes` map, not a fixed enum.
