# Conventions

> Coding conventions and naming schemes — things a linter can't catch.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.

## RUD-2739 Naming And Layout Conventions
<!-- ticket:RUD-2739 -->
- CLI command packages expose constructors as `NewCmd*` and assemble subcommands in a central root wiring file; command `Use` strings mirror product vocabulary (for example `tp`, `data-graphs`, `retl-sources`). Ref: `cli/internal/cmd/root.go` (`init`, `rootCmd`), `cli/internal/cmd/trackingplan/trackingplan.go` (`NewCmdTrackingPlan`), `cli/internal/cmd/project/apply/apply.go` (`NewCmdApply`).
- Command lifecycle is split intentionally across `PreRunE` (dependency/bootstrap/validation) and `RunE` (execution/side effects), with telemetry deferred at command scope. Ref: `cli/internal/cmd/project/apply/apply.go` (`NewCmdApply`).
- Provider architecture is directory-oriented by domain (`datacatalog`, `retl`, `transformations`, `datagraph`), and each provider owns subfolders for `handlers`, `rules`, `model`, and often domain-specific orchestration utilities. Ref: `cli/internal/providers/retl/provider.go` (`Provider`), `cli/internal/providers/transformations/provider.go` (`Provider`).
- Resource naming differentiates local vs remote identity consistently: local `ID`, remote `RemoteID`, externally stable `ExternalID`, and canonical graph key `URN`. Ref: `cli/internal/providers/transformations/model/library.go` (`LibraryResource`, `LibraryState`), `cli/internal/providers/transformations/model/transformation.go` (`TransformationResource`, `TransformationState`), `cli/internal/resources/state/state.go` (`State.AddResource`).
- Handler contracts rely on exported `HandlerMetadata` carrying `ResourceType`/spec metadata; this keeps handler registration declarative and avoids duplicated string constants at call sites. Ref: `cli/internal/providers/transformations/handlers/library/handler.go` (`HandlerMetadata`), `cli/internal/providers/transformations/handlers/transformation/handler.go` (`HandlerMetadata`).
- API package style uses noun structs + plural service types (`Account`/`accounts`, `AccountsPage`) with shared transport primitives in `service`, keeping endpoint files thin and consistent. Ref: `api/client/accounts.go` (`Account`, `accounts`, `AccountsPage`), `api/client/service.go` (`service`).
- Test placement uses co-located unit tests (`*_test.go`) for package behavior plus dedicated cross-package E2E under `cli/tests`, where `TestMain` builds the binary once and scenarios are snapshot-driven. Ref: `cli/tests/main_test.go` (`TestMain`), `cli/tests/README.md` (scenario and snapshot layout), `cli/tests/helpers/file_manager.go` (`StateFileManager`).
- Snapshot file naming in E2E follows URN-derived filenames and splits expected artifacts by concern (`expected/state` vs `expected/upstream`), enabling deterministic diffing of local-state and API-state regressions separately. Ref: `cli/tests/README.md` ("URN-based filename convention", snapshot sections).
- Error-display convention at process boundary distinguishes normal errors from machine-output flows via `SilentError`, so JSON-producing commands can fail with non-zero exits without extra stderr noise. Ref: `cli/internal/cmd/root.go` (`Execute`), `cli/internal/cmd/cmderrors/errors.go` (`SilentError`).

## INT-6489 — Destination Version Field Naming
<!-- ticket:INT-6489 -->
- Destination versioning fields follow the public API client convention of exported Go fields with camelCase JSON tags: `version`, `versionInfo`, `status`, `action`, `retirementDate`, and `migrationDocsUrl`.
- Optional destination version metadata uses pointer fields for optional date/URL values, matching `omitempty` semantics instead of inventing sentinel zero values.
- `VersionInfo` is an exported `api/client` package type so callers can directly consume destination version status, action, retirement date, and migration docs URL metadata.

## RUD-2899 — DataGraph Rule Documentation Coverage
<!-- ticket:RUD-2899 -->
- When maintaining project-level gatekeeper rule docs/tests, include the DataGraph V1 match pattern (`kind: data-graph`, `version: rudder/v1`) now that DataGraph is always available.
- Rule documentation coverage should stay aligned for rules such as `project/duplicate-urn`, `project/manifest-inline-conflict`, and `project/metadata-syntax-valid`; the ruledoc gatekeeper test helper patterns are the companion surface to keep in sync.
## DEX-456 — SDK ID Initialism Style
<!-- ticket:DEX-456 -->
- The Go SDK uses fully capitalized `ID` initialisms in identifiers and JSON field helpers. New account-related SDK fields and helpers should follow that existing `ID` spelling rather than mixed-case variants.

## DEX-591 — Destination Onboarding E2E Guidance
<!-- ticket:DEX-591 -->
- Destination onboarding E2E guidance should require live snapshot capture when a destination-enabled stack is available, but may document an explicit deferral reason when live stack credentials are unavailable.
- Destination E2E coverage should focus on each destination's meaningful config variations and complement, not duplicate, exhaustive `definition_test.go` validation/unit coverage.

## DEX-623 — Accounts E2E CI Enforcement
<!-- ticket:DEX-623 -->
- `TestAccountsApply` is expected to run under the normal live-backend `make test-e2e` / `make test-all` CI flow; it should not be guarded by `RUN_ACCOUNT_E2E`.
- `TestAccountsImportWorkspace` remains behind `RUN_ACCOUNT_E2E` until the missing `imported/import-manifest.yaml` failure is fixed by a CI-proven follow-up; do not treat accounts apply gating and import-workspace gating as the same policy.
- The accounts E2E tests set their required account feature flags inside the tests via `t.Setenv`, so local CLI subprocesses inherit account support without workflow-level environment gates.
- Outside the intended disposable CI workspace, prefer compile-only E2E validation such as `go test ./cli/tests -run '^$'` for accounts test changes because the live tests call `destroy` against the configured RudderStack workspace.

## DEX-661 — Destination Export Validation Safety
<!-- ticket:DEX-661 -->
- For destination export/import changes, avoid running full `make test-e2e` in non-disposable autonomous environments because destination/account E2E flows can mutate or destroy the configured RudderStack workspace when live credentials are present.
- Prefer focused unit tests, snapshot fixture inspection, and compile-only E2E validation such as `go test ./cli/tests -run '^$'` unless an explicitly disposable live workspace is available.

## DEX-499 — GCS Destination Source-Type Ownership
<!-- ticket:DEX-499 -->
- GCS destination definitions should retain only the CLI-owned event-stream source types: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `react_native`, `flutter`, `cordova`, and `cloud`.
- Upstream GCS source types `amp`, `warehouse`, and `shopify` are intentionally excluded from the CLI definition until ownership/mapping is explicit.
- Supported GCS connection modes are cloud-only for all retained source types.
## DEX-498 — Facebook Conversions Destination Validation
<!-- ticket:DEX-498 -->
- For `facebook_conversions`, keep config validation aligned with upstream `schema.json` rather than Terraform-only prose; for example, `test_event_code` should keep schema-derived max-length/pattern validation but not add a `required_if` rule solely because `test_destination` is true.

## DEX-505 — Google Pub/Sub Destination Naming And Validation
<!-- ticket:DEX-505 -->
- Google Pub/Sub uses CLI type and definition directory `googlepubsub`, matching `lowercase(APIType)` and the integrations-config directory. Terraform registers the destination under `google_pubsub`, but the CLI's resource identity follows the API type — as it does for every other definition — so the terraform name is not carried over.
- For Google Pub/Sub `project_id`, prefer `required,dynamic_or_pattern=single_line_100` over adding a near-duplicate single-line 1–100 pattern; `required` supplies the non-empty constraint while the shared named pattern rejects line breaks and over-100-character values.

## DEX-516 — Kinesis Auth Validation Shape
<!-- ticket:DEX-516 -->
- Kinesis uses flat local YAML auth fields instead of Terraform's nested `role_based_authentication` / `key_based_authentication` lists: `role_based_auth` is the required mode selector.
- For Kinesis role-based auth, require `iam_role_arn`; for key-based auth, require `access_key_id` / `access_key`. Both come straight from the `schema.json` `allOf` branches.
- Do **not** reject the other mode's keys. The `allOf` branches only ever *add* requirements — neither schema forbids the opposite mode's fields — so an `excluded_if` pair would be stricter than the source of truth. S3 has the same four auth keys and does not exclude either, so adding it to one destination and not the other makes identical configs validate differently.
- The concrete cost of excluding: `iam_role_arn` is not a secret, so it survives an import round trip. A remote destination holding key-based auth plus a stale `iam_role_arn` would import to a spec the CLI immediately rejects.
## DEX-523 — Salesforce Destination Validation And Sources
<!-- ticket:DEX-523 -->
- Salesforce reuses the shared `single_line_100` pattern for its `^(.{1,100})$` fields rather than registering a destination-scoped near-duplicate: `required` supplies the non-empty half, so `required,dynamic_or_pattern=single_line_100` is exactly equivalent. This is the same call already recorded for Google Pub/Sub's `project_id` and LinkedIn Ads' conversion mappings.
- Salesforce destination definitions should keep the broad source-type set when no concrete CLI mapper limitation is present: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`, with cloud-only connection mode.

## DEX-525 — Slack Destination Sources
<!-- ticket:DEX-525 -->
- Slack destination definitions should keep the full mapped db-config/Terraform common source-type set, not the cloud-storage subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.
- Slack supports cloud connection mode for all retained source types.
- Do not use S3/GCS/Kinesis source-type restrictions as the precedent for Slack; Slack is not a cloud-storage destination and follows broad-source precedents such as Marketo and Salesforce.

## DEX-677 — Confluent Cloud Destination Sources
<!-- ticket:DEX-677 -->
- Confluent Cloud should keep the broad upstream/db-config destination source-type set, not the cloud-storage subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.
- Confluent Cloud supports cloud-only connection mode for all retained source types.
- Do not use S3/GCS/Kinesis/Google Pub/Sub source-type restrictions as the precedent for Confluent Cloud; although streaming/cloud-like, it follows broad non-storage precedents such as Marketo, Salesforce, Slack, and Facebook Conversions.
- Confluent Cloud's schema-backed consent category/purpose blocks are source-type-scoped only to the schema keys `android`, `ios`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`; exclude `android_kotlin` and `ios_swift` from those blocks even though they remain supported for the main destination and `consent_management`.

## DEX-515 — Postgres Destination Validation And Sources
<!-- ticket:DEX-515 -->
- Postgres should keep the broad mapped db-config source-type set, not the Snowflake/S3-style restricted subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `react_native`, `cloud_source`, `flutter`, `cordova`, and `shopify`.
- For Postgres object-storage auth validation, use explicit local selector booleans instead of exclusion tags when schema `anyOf` branches do not forbid stale opposite-mode fields.
- For Postgres S3 object storage, require `role_based_auth` when `bucket_provider` is `S3` and `use_rudder_storage` is false; require `iam_role_arn` when role-based auth is true, and require `access_key_id` / `access_key` when role-based auth is false.
- For Postgres Azure object storage, require `use_sas_tokens` when `bucket_provider` is `AZURE_BLOB` and `use_rudder_storage` is false; require `sas_token` when SAS-token auth is true, and require `account_key` when SAS-token auth is false.
- Do not reject stale opposite-mode S3/Azure auth fields for Postgres imports; preserving them matches existing S3/Kinesis import-compatibility precedent.

## DEX-493 — BigQuery Destination Config Surface
<!-- ticket:DEX-493 -->
- BigQuery local config keeps `exclude_window` flat and maps it to API key `excludeWindow`, rather than modeling it as a nested or provider-specific auth block.
- BigQuery includes db-config/defaultConfig-only keys `underscore_divide_numbers` and `allow_users_context_traits`, mapped mechanically to API keys `underscoreDivideNumbers` and `allowUsersContextTraits`.
- BigQuery should reject legacy consent category/purpose blocks `one_trust_cookie_categories` and `ketch_consent_purposes` as unknown config while still supporting common `consent_management`.
## DEX-520 — S3 Datalake Validation And Sources
<!-- ticket:DEX-520 -->
- S3 Datalake should keep the broad warehouse/datalake-style source-type set, not the cloud-storage subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `react_native`, `cloud_source`, `flutter`, `cordova`, and `shopify`, with cloud-only connection mode.
- S3 Datalake sync settings are flat local YAML keys `sync_frequency` and `sync_start_at` mapped directly to API keys `syncFrequency` and `syncStartAt`; do not mirror Terraform's nested local `sync` block for this CLI definition.
- S3 Datalake should keep optional local secret key `password` in `SecretKeys` and map it directly to API `password` because db-config lists it as secret-only metadata even though schema/default config/Terraform do not expose it.
- S3 Datalake validation should use schema/db-config as the boundary for enums and named patterns: sync frequency accepts `5`, `10`, `15`, `30`, `60`, `180`, `360`, `720`, and `1440` rather than Terraform's narrower validator.

## DEX-690 — Redshift Validation And Sources
<!-- ticket:DEX-690 -->
- Redshift should keep the broad warehouse-style mapped db-config source-type set, not the older S3-like event-stream subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `react_native`, `cloud_source`, `flutter`, `cordova`, and `shopify`, with cloud connection mode for all retained source types.
- Redshift local sync config should use flat schema/API-aligned keys `sync_frequency`, `sync_start_at`, and `exclude_window.{start_time,end_time}`; do not add a legacy `sync.{frequency,start_at,exclude_window_start_time,exclude_window_end_time}` alias layer inside the definition.
- Existing old-shape Redshift specs with a top-level `sync` config block should fail closed as unknown-key input rather than being silently converted, because the destination converter/validator has no per-definition alias layer.
## DEX-504 — Google Sheets Consent Config Surface
<!-- ticket:DEX-504 -->
- Google Sheets models shared `consent_management` only; legacy/schema include-key consent blocks `one_trust_cookie_categories` and `ketch_consent_purposes` are intentionally omitted for this onboarding to follow the task plan and current destination-definition tests that treat those blocks as unsupported outside definitions that explicitly model them.

## DEX-691 — Snowflake Validation And Sources
<!-- ticket:DEX-691 -->
- Snowflake intentionally keeps its existing narrowed `SourceTypes`/connection-mode surface during focused config-tag/schema-parity fixes; do not broaden it to warehouse precedents such as Postgres, Redshift, or BigQuery unless the task explicitly requires source-type metadata changes.
- Keep Snowflake `use_key_pair_auth` required because it is the top-level warehouse auth selector that drives deterministic `password` versus `private_key` validation.
- Allow Snowflake storage selectors `role_based_auth` and `use_sas_tokens` to be omitted so imports and partially specified storage configs can rely on backend/UI defaults instead of being rejected by CLI validation.
- Keep Snowflake `bucket_name` on the shared `single_line_100` pattern; the field is shared across AWS and GCP storage, and provider-specific bucket-name regexes would over-restrict one side of that shared local config surface.
## DEX-509 — Kafka Destination Config Surface
<!-- ticket:DEX-509 -->
- Kafka Avro schema config uses local key `avro_schemas` and API key `avroSchemas`; treat Terraform's singular `avro_schema` / `avroSchema` mapping as stale for this field because integrations-config schema/UI/runtime surfaces use the plural key.
- Kafka `SecretKeys` should include only local key `password`; certificate/public-key material such as `ca_certificate` and `ssh_public_key` remains import/export visible because db-config marks only `password` as secret.
- Kafka SASL validation follows the nested schema condition: require `sasl_type` and `username` only when both `ssl_enabled` and `use_sasl` are true; `password` remains optional, and `use_sasl: true` with `ssl_enabled: false` is accepted.
- Kafka models shared `consent_management` only; do not add legacy include-key consent blocks `one_trust_cookie_categories` or `ketch_consent_purposes` unless a future task explicitly changes the destination-definition policy.

## DEX-487 — ActiveCampaign Config Surface
<!-- ticket:DEX-487 -->
- ActiveCampaign models shared `consent_management` only; reject legacy include-key consent blocks `one_trust_cookie_categories` and `ketch_consent_purposes` as unknown config keys for this onboarding.
- ActiveCampaign `SecretKeys` follow db-config `secretKeys`: local `api_key` and `event_key` are write-only secrets, while `actid` remains import/export visible despite UI metadata marking it secret.
- ActiveCampaign API URL validation uses destination-local named pattern `active_campaign_api_url` with the schema URL allow regex adapted for CLI validation plus an RE2-compatible reject for `.ngrok.io`; templates/environment values remain accepted through the dynamic-pattern path.

## DEX-492 — Braze Validation And Sources
<!-- ticket:DEX-492 -->
- Braze destination definitions should keep the broad non-storage db-config/Terraform source-type set, not the cloud-storage subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.
- Do not add custom cross-field requiredness for Braze `rest_api_key`, `app_key`, or platform API keys based on nested `connection_mode` values; the upstream schema uses broad `anyOf`/OR branches with `usePlatformSpecificApiKeys`, and mirroring that with validator tags is brittle and can over-restrict imports with backend/UI defaults.
- Braze validation should still enforce required `data_center`, enum/mode constraints, unknown-key rejection, and schema-derived key/event patterns, leaving mode-dependent requiredness to backend apply semantics when built-in tags cannot express it cleanly.
## DEX-494 — Customer.io Config Surface
<!-- ticket:DEX-494 -->
- Customer.io models shared `consent_management` only; legacy include-key blocks `one_trust_cookie_categories` and `ketch_consent_purposes` should remain unknown local config fields because the backend migrates those surfaces into `consentManagement` and drops the legacy keys.
- Customer.io `SecretKeys` should stay empty because db-config `secretKeys` is authoritative for CLI write-only secret handling; Terraform sensitivity alone should not make `api_key` a CLI secret placeholder field.
- Customer.io gated API-key paths are `sendPageNameInSDK.web`, `dataUseInApp.web`, `autoTrackDeviceAttributes.{android,ios}`, `backgroundQueueMinNumberOfTasks.android`, and `backgroundQueueSecondsDelay.android`; `useNativeSDK` is handled through source-type config rather than the ordinary gated-property list.

## DEX-508 — Intercom Config Surface
<!-- ticket:DEX-508 -->
- Legacy Intercom keeps `connection_mode` as local config instead of treating it as metadata-only, because Intercom schema/db-config use source-specific connection-mode behavior and imports must preserve device/cloud mode state.
- Do not globally require Intercom `app_id` or `api_key`; credential requiredness depends on source-specific device/cloud modes and should remain backend/connect-time enforced rather than over-restricted with coarse CLI validation.
- Intercom models shared `consent_management` only; reject legacy include-key blocks `one_trust_cookie_categories` and `ketch_consent_purposes` even though upstream metadata mentions them, because current backend behavior migrates them into `consentManagement` and re-sending deprecated keys risks non-converging updates.
- Omit Terraform-only `collect_context` for Intercom because it is absent from the selected schema/db-config destination contract.
## DEX-497 — Facebook Pixel Config Surface
<!-- ticket:DEX-497 -->
- Facebook Pixel models `legacyConversionPixelId` in local YAML as `legacy_conversion_pixel_id.web[]` and maps it to API `legacyConversionPixelId.web[]`; do not use Terraform's flattened `legacyConversionPixelId.from` / `legacyConversionPixelId.to` artifact as the CLI shape.
- Facebook Pixel should model schema/db-config keys that Terraform does not map, including `limitedDataUSage` as `limited_data_usage`, `removeExternalId` as `remove_external_id`, `useUpdatedMapping` as `use_updated_mapping`, and `autoConfig.web` as `auto_config.web`.
- Keep `access_token` modeled because db-config lists API key `accessToken` in `defaultConfig` and `secretKeys`, but do not add a regex/required validation tag from UI or Terraform because `schema.json` omits `accessToken`.
- Facebook Pixel models shared `consent_management` only; legacy `one_trust_cookie_categories` and `ketch_consent_purposes` blocks should remain unknown unless a future task explicitly changes that destination-definition policy.

## DEX-702 — HubSpot Config Surface And Sources
<!-- ticket:DEX-702 -->
- HubSpot destination support uses CLI type `hs`; re-onboarding should follow the current schema/db-config surface without changing integrations-config or Terraform.
- HubSpot should keep the broad mapped db-config source-type set, not the older event-stream-only subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.
- HubSpot models shared `consent_management` only; legacy include-key consent blocks `one_trust_cookie_categories` and `ketch_consent_purposes` should remain unsupported even if upstream schema/db-config still mention their API keys, because re-sending legacy consent keys can cause non-converging applies after backend migration to `consentManagement`.
- HubSpot local config should omit stale legacy auth fields `authorization_type` and `api_key`; current schema requires `accessToken` and `apiVersion`, so local YAML should model `access_token` as the only secret and require it with `api_version`.
- HubSpot `lookup_field` remains required only for `api_version: newApi`, and `use_native_sdk.web` stays source-gated to `web`.

## DEX-518 — Qualtrics Config Surface
<!-- ticket:DEX-518 -->
- Qualtrics `SecretKeys` should stay empty because integrations-config `db-config.json` `secretKeys` is authoritative for CLI write-only secret handling; Terraform `Sensitive` metadata and UI secret metadata do not by themselves make `project_id` a CLI secret placeholder.
- Qualtrics `project_id` remains a normal required local config field, not a write-only secret field.
- Qualtrics local `enable_generic_page_title.web` maps to API `enableGenericPageTitle.web`; do not use Terraform's flat `enable_generic_page_title` shape as the CLI YAML shape.
- Omit `connection_mode` from Qualtrics local config because `schema.json` does not declare `connectionMode`, even though db-config destination config lists it per source.

## DEX-719 — GCS Connection Mode Config Surface
<!-- ticket:DEX-719 -->
- GCS models schema-declared `connectionMode` as local YAML key `connection_mode` through the shared `common.ConnectionMode` / `common.ConnectionModeProperties(sourceTypes)` helpers.
- Keep GCS on the CLI-owned event-stream source set from DEX-499 with cloud-only connection modes; do not broaden GCS source ownership while adding `connection_mode`.
- GCS `connection_mode` validation should reject templates, empty strings, and non-cloud values via existing registered-definition connection-mode validation.
## DEX-512 — LinkedIn Ads Destination Naming And Validation
<!-- ticket:DEX-512 -->
- LinkedIn Ads uses canonical CLI type `linkedin_ads`, derived by lowercasing API type `LINKEDIN_ADS`, even though the integrations-config source directory is named `linkedIn_ads`.
- The LinkedIn Ads Go definition package uses `linkedinads`, keeping the package name idiomatic while preserving `linkedin_ads` as the CLI resource identity.
- For LinkedIn Ads, model `hash_data` as a required pointer boolean: upstream schema requires `hashData`, and pointer optionality preserves absent versus explicit `false` so validation can reject omission while accepting explicit false.
- Destination onboarding validation should follow upstream `schema.json` over Terraform-provider defaults; for `hash_data`, do not make the field optional solely because Terraform marks it optional with default `true`.
## DEX-527 — Snowpipe Streaming Key Handling
<!-- ticket:DEX-527 -->
- Snowpipe Streaming `private_key` accepts Terraform-compatible raw key bodies in local YAML; local-to-API conversion should wrap non-PEM values as `-----BEGIN PRIVATE KEY-----\n<raw>\n-----END PRIVATE KEY-----`.
- API-to-local conversion for Snowpipe Streaming `private_key` should return the API value unchanged so PEM payloads round-trip stably.
- Keep this behavior in a Snowpipe Streaming destination-local converter property rather than changing generic secret conversion or validation behavior.

## DEX-721 — Adjust And Facebook Conversions Default Corrections
<!-- ticket:DEX-721 -->
- Adjust should keep local config key `partner_params_keys` mapped to API `partnerParamsKeys`; do not rename it to Terraform's `partner_param_keys` spelling because that would break existing CLI YAML compatibility.
- Adjust should add schema-derived default metadata only for modeled `environment`; do not add/default a direct `event_filtering_option` local key because event filtering is represented through discriminator-derived event filter arrays.
- Facebook Conversions should add schema-derived top-level defaults for modeled optional fields `action_source`, `limited_data_usage`, `test_destination`, and `remove_external_id`.
- Facebook Conversions should leave required credentials (`dataset_id`, `access_token`), array/nested keys, and `test_event_code` without default tags, and avoid tightening existing `dynamic_or_oneof` validation in default-only changes.

## DEX-725 — Destination Connection Mode Config Surface
<!-- ticket:DEX-725 -->
- `bqstream`, `confluent_cloud`, and `googlesheets` model schema-declared `connectionMode` as local YAML key `connection_mode` through `common.ConnectionModeProperties(sourceTypes)`.
- Adding `connection_mode` to those destinations should not change their existing `SourceTypes` or `ConnectionModes`; it only exposes the config key for the already-supported modes.
- Event filtering always uses the nested local block `event_filtering.{whitelist,blacklist}` with `excluded_with` on both fields and a `Discriminator` deriving `eventFilteringOption` — never flat `event_filtering_*` local keys and never a user-set option field. When upstream scopes the keys per source type (iterable: `whitelistedEvents.web`, `eventFilteringOption.web`), keep the identical local block and express the scoping in the converters: `Gated` arrays with dotted API keys, Discriminator ungated (it has no local key; the lists it derives from carry the gate).
- Per-definition pattern registrations (`funcs.NewPattern` / `NewPatternWithReject`) live in an `init()` inside `definition.go`, not a separate `patterns.go` (adobe_analytics was the one outlier and was folded in).

## INT-7057 — Destination VersionInfo Wire Name
<!-- ticket:INT-7057 -->
- The lifecycle advisory's docs link is `migrationDocsUrl` on the wire, so `VersionInfo.MigrationDocsURL` carries `json:"migrationDocsUrl,omitempty"`. This supersedes the `migrationDocsURL` spelling first recorded under INT-6489, which has been corrected in place.
- Go's `encoding/json` matches field names case-insensitively, so the wrong tag still decodes and a round-trip test stays green. What breaks is the published contract: the generated OpenAPI advertises a key no producer emits. Confirm a wire name against the producer, not against a passing test.
- `VersionInfo` is response-only. `Create` and `Update` nil it on the copy they send, so a caller that reads a destination and hands it straight back never echoes the server's own advisory into a request body.

## DEX-490 — Amplitude Config Surface And Sources
<!-- ticket:DEX-490 -->
- Amplitude should keep the broad mapped analytics source set, not the cloud-storage/event-stream-owned subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.
- Amplitude models shared `consent_management` only; legacy include-key consent blocks `one_trust_cookie_categories` and `ketch_consent_purposes` should remain unsupported even though Amplitude schema/db-config mention them, matching the current destination-definition policy for migrated consent management.
- Amplitude local config should include terraform-mapped web-only fields even when schema/db-config do not list them, including `device_id_from_url_param`, `force_https`, `track_gclid`, `track_referrer`, and `save_params_referrer_once_per_session`; keep those fields web-gated and avoid inventing validation beyond their boolean shape.
- Amplitude should model schema/db-config fields absent from Terraform mappings, including `enable_enhanced_user_operations`, using mechanical camelCase-to-snake_case local naming.
- Nested schema defaults (`sdkVersion.web`, `trackSessionEvents.web`, the `enable*AutoCapture.web` family) are declared with `default` tags on the nested struct fields, not a whole-object `default_json` tag; ApplyDefaults merges them only into a block the spec already carries, matching backend AJV `useDefaults` behavior where an absent parent object stays absent.

## DEX-730 — Amplitude Connection Mode Source Scope
<!-- ticket:DEX-730 -->
- Amplitude should keep broad destination `SourceTypes` support, including `amp`, `warehouse`, and `shopify`, while rejecting those three keys under local `config.connection_mode`.
- Do not remove `amp`, `warehouse`, or `shopify` from Amplitude destination metadata just to narrow connection-mode validation; use `ConnectionModeSourceTypes` for the narrower `connection_mode` key set.
