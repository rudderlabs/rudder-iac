# Architecture

> Component layout, internal relationships, data flow.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.

## Runtime Composition and Dependency Injection
<!-- ticket:RUD-2739 -->
- CLI bootstrap is layered: the binary entrypoint sets version and delegates command execution, and the root command initializes config/logger/app dependencies before subcommands run (`cli/cmd/rudder-cli/main.go::main`, `cli/internal/cmd/root.go::Execute`, `cli/internal/cmd/root.go::init`).
- Dependency assembly is centralized so commands use one construction path for API client + providers + composite provider, avoiding per-command wiring drift (`cli/internal/app/dependencies.go::NewDeps`, `cli/internal/app/dependencies.go::setupClient`, `cli/internal/app/dependencies.go::setupProviders`).
- Provider composition is feature-gated: DataGraph is conditionally inserted into the provider map, so command visibility and graph capability expand together under the same experimental flag (`cli/internal/cmd/root.go::initConfig`, `cli/internal/app/dependencies.go::NewDeps`).
- The app separates "single-provider" vs "multi-provider" project scopes: tracking-plan-only flows can run against DataCatalog alone, while general project flows use the composite provider (`cli/internal/app/dependencies.go::NewDataCatalogProject`, `cli/internal/app/dependencies.go::NewProject`).
- Type/kind dispatch is explicit contract surface: providers advertise `SupportedKinds`, `SupportedTypes`, and `SupportedMatchPatterns`, then routing is resolved via kind for spec loading and type for lifecycle operations (`cli/internal/provider/provider.go::TypeProvider`, `cli/internal/provider/composite.go::providerForKind`, `cli/internal/provider/composite.go::providerForType`).

## Spec Loading and Validation Pipeline
<!-- ticket:RUD-2739 -->
- Project loading is a two-phase pipeline: read raw YAML files recursively, parse strict specs, then run syntax and semantic validations in order to fail fast before graph-heavy checks (`cli/internal/project/loader/loader.go::Load`, `cli/internal/project/project.go::Load`, `cli/internal/project/project.go::handleValidation`).
- Strict spec parsing rejects unknown top-level fields by design, which makes rule-based validation operate on normalized documents rather than permissive inputs (`cli/internal/project/specs/spec.go::New`).
- Validation registry merges global gatekeeper rules with provider-contributed rules, making providers the extension seam for domain constraints while keeping a shared orchestrator (`cli/internal/project/project.go::registry`, `cli/internal/provider/provider.go::RuleProvider`, `cli/internal/provider/baseprovider.go::SyntacticRules`, `cli/internal/provider/baseprovider.go::SemanticRules`).
- Resource graph construction happens after syntax checks and before semantic checks, so semantic rules can reason over cross-resource structure on a single graph source of truth (`cli/internal/project/project.go::handleValidation`, `cli/internal/provider/provider.go::SpecLoader::ResourceGraph`).
- Cycle detection is enforced before semantic validation completes, preventing downstream planning on invalid dependency topology (`cli/internal/project/project.go::handleValidation`, `cli/internal/resources/graph.go::DetectCycles`).
- Commands reuse the same project-load path: `validate` only runs the pipeline, while `apply` continues into sync after the same load+validate gate (`cli/internal/cmd/project/validate/validate.go::NewCmdValidate`, `cli/internal/cmd/project/apply/apply.go::NewCmdApply`).

## Graph Model and Reference Resolution
<!-- ticket:RUD-2739 -->
- `resources.Graph` is the dependency backbone, storing resources plus both dependency and dependent adjacency maps for forward and reverse traversal (`cli/internal/resources/graph.go::Graph`).
- Dependencies are inferred from embedded references (`PropertyRef`) found in map-based data and reflected raw structs, so typed handlers can still participate in graph-level ordering without flattening all resource models (`cli/internal/resources/graph.go::AddResource`, `cli/internal/resources/references.go::collectReferences`, `cli/internal/resources/references.go::collectReferencesByReflection`).
- Resource identity is globally normalized as `type:id` URNs; this same key threads through graph nodes, state entries, import metadata, and diff/planner operations (`cli/internal/resources/resource.go::URN`, `cli/internal/resources/state/state.go::AddResource`).
- State stores both generic maps and typed raw payloads, enabling dual execution paths: map dereference for generic providers and reflection-based dereference for typed/raw providers (`cli/internal/resources/state/state.go::ResourceState`, `cli/internal/resources/state/state.go::Dereference`, `cli/internal/syncer/syncer.go::createOperation`).
- Reference dereferencing is recursive and state-backed, so runtime inputs can resolve transitive references to current state outputs before CRUD/import calls (`cli/internal/resources/state/state.go::dereferenceValue`, `cli/internal/syncer/syncer.go::updateOperation`).

## Diff, Planning, and Apply Execution
<!-- ticket:RUD-2739 -->
- Apply flow computes a remote-backed source graph, diffs it against local target graph, builds ordered operations, then executes CRUD/import through provider lifecycle methods (`cli/internal/syncer/syncer.go::apply`, `cli/internal/syncer/planner/planner.go::Plan`, `cli/internal/syncer/syncer.go::providerOperation`).
- Diff classification encodes import semantics: target-only resources with matching workspace import metadata become `Import` operations, while other target-only resources become `Create` (`cli/internal/syncer/differ/diff.go::ComputeDiff`).
- Planner ordering is dependency-aware: creates/imports/updates run in dependency order, and deletes reverse dependency order to avoid removing prerequisites too early (`cli/internal/syncer/planner/planner.go::Plan`, `cli/internal/syncer/planner/planner.go::sortByDependencies`).
- Concurrent execution honors dependency edges via task graph waiting; delete tasks invert dependency lookup by using source-graph dependents (`cli/internal/syncer/operation_task.go::Dependencies`, `cli/pkg/tasker/task.go::RunTasks`).
- Execution mutates in-memory state incrementally under lock, so later tasks can dereference outputs from earlier successful operations in the same run (`cli/internal/syncer/syncer.go::createOperation`, `cli/internal/syncer/syncer.go::importOperation`, `cli/internal/syncer/syncer.go::updateOperation`).
- Post-plan consolidation is a provider hook for cross-resource finalization after individual operations complete (`cli/internal/provider/provider.go::ConsolidateSyncer`, `cli/internal/syncer/syncer.go::apply`).

## Import and Export Data Flow
<!-- ticket:RUD-2739 -->
- Workspace import is gated on "no pending drift": it compares remote-derived source graph and local target graph and aborts if diff exists, preventing mixed reconciliation+import in one step (`cli/internal/project/importer/importer.go::WorkspaceImport`, `cli/internal/project/importer/importer.go::ErrProjectNotSynced`).
- Import ID allocation preloads existing project IDs into a scoped namer to avoid collisions when generating new imported specs (`cli/internal/project/importer/importer.go::initNamer`).
- Reference rewriting during export resolves links first against to-be-imported resources, then against already-managed remote resources mapped through graph file metadata (`cli/internal/resolver/resolver.go::ImportRefResolver::ResolveToReference`).
- Export is provider-driven and returns `FormattableEntity` records, while write/format responsibilities are centralized and extension-based by file suffix (`cli/internal/provider/provider.go::Exporter`, `cli/internal/project/writer/write.go::Write`, `cli/internal/project/formatter/formatter.go::Formatters::Format`).
- File emission is intentionally non-destructive (`O_EXCL`) for import outputs, aligning with the command-level check that `imported/` must not preexist (`cli/internal/project/writer/write.go::writeFile`, `cli/internal/cmd/import/workspace.go::NewCmdWorkspaceImport`).

## Provider Specializations and Cross-Resource Semantics
<!-- ticket:RUD-2739 -->
- `BaseProvider` implements the common handler fan-out pattern (spec load, graph build, remote load, state map, CRUD/import routing), so most domain providers only declare handlers and overrides where needed (`cli/internal/provider/baseprovider.go::NewBaseProvider`, `cli/internal/provider/baseprovider.go::ResourceGraph`).
- DataCatalog uses an internal local-catalog model and explicit graph synthesis with metadata/import options, including tracking-plan dependency projection to property/event URNs (`cli/internal/providers/datacatalog/provider.go::ResourceGraph`, `cli/internal/providers/datacatalog/provider.go::createResourceGraph`).
- Transformations provider overrides graph/state semantics to derive transformation-library dependencies from code imports and remote import lists, then uses consolidate-sync for batch publish and deferred deletes (`cli/internal/providers/transformations/provider.go::ResourceGraph`, `cli/internal/providers/transformations/provider.go::MapRemoteToState`, `cli/internal/providers/transformations/provider.go::ConsolidateSync`, `cli/internal/providers/transformations/provider.go::DeleteRaw`).
- DataGraph provider extends base loading/parsing for composite inline specs (data-graph + models + relationships in one document), and export reconstructs that composite shape from grouped remote resources (`cli/internal/providers/datagraph/provider.go::ParseSpec`, `cli/internal/providers/datagraph/provider.go::LoadSpec`, `cli/internal/providers/datagraph/provider.go::FormatForExport`).
- DataGraph validation has its own orchestration path layered on top of project graph + remote diff: mode-based planning (all/modified/single), account resolution from parent graph resources, then concurrent validation task execution (`cli/internal/cmd/datagraph/validate/validate.go::NewCmdValidate`, `cli/internal/providers/datagraph/validator/validator.go::Validate`, `cli/internal/providers/datagraph/validator/runner.go::Run`, `cli/internal/providers/datagraph/validator/planner.go::PlanModified`).
- Composite provider parallelizes cross-provider remote/importable loading using the shared task executor, then merges collections/states, giving multi-domain commands a unified state/diff surface (`cli/internal/provider/composite.go::LoadResourcesFromRemote`, `cli/internal/provider/composite.go::LoadImportable`, `cli/internal/provider/composite.go::MapRemoteToState`).

## Cross-cutting
<!-- ticket:RUD-2739 -->

- Bootstrap centralization is both an intentional composition pattern and a scaling risk: root/bootstrap files and dependency wiring define a single DI path, but that same concentration is flagged as coupling/god-object drift as providers/features grow (`cli/cmd/rudder-cli/main.go::main`, `cli/internal/cmd/root.go::init`, `cli/internal/app/dependencies.go::NewDeps`) — see `entry-points.md` and `concerns.md`.
- The dependency-ordered execution model is a repo-wide invariant: graph-derived ordering drives sync planning and task execution, and the same task runtime semantics surface in recurring concurrency patterns (`cli/internal/resources/graph.go::Graph`, `cli/internal/syncer/planner/planner.go::Plan`, `cli/pkg/tasker/task.go::RunTasks`) — see `patterns.md` and `architecture.md`.
- Provider extensibility is achieved via large shared contracts plus base-provider fan-out, which accelerates new domains but also appears in interface-size and composite-provider smell reports (`cli/internal/provider/provider.go::Provider`, `cli/internal/provider/baseprovider.go::NewBaseProvider`, `cli/internal/provider/composite.go::CompositeProvider`) — see `conventions.md` and `concerns.md`.
- State is intentionally dual-form (`map` + typed raw) to support generic/typed providers and recursive reference resolution during apply; this same flexibility increases correctness pressure around synchronized mutation and dereference timing (`cli/internal/resources/state/state.go::ResourceState`, `cli/internal/resources/state/state.go::Dereference`, `cli/internal/syncer/syncer.go::createOperation`) — see `patterns.md` and `architecture.md`.
- Strict load/validate-before-apply gating is consistent from command lifecycle conventions through project pipeline architecture, forming a shared safety boundary for both `validate` and `apply` entrypoints (`cli/internal/cmd/project/apply/apply.go::NewCmdApply`, `cli/internal/project/project.go::Load`, `cli/internal/project/project.go::handleValidation`) — see `conventions.md` and `entry-points.md`.
- Import/export flows prioritize safety and determinism (non-destructive writes, sync precondition checks, formatter routing), yet unresolved TODOs in provider export/import handlers show uneven maturity across domains (`cli/internal/project/importer/importer.go::WorkspaceImport`, `cli/internal/project/writer/write.go::writeFile`, `cli/internal/providers/transformations/handlers/library/handler.go::Export`) — see `architecture.md` and `concerns.md`.
- Observability and failure semantics are deliberately structured (wrapped errors, task failure typing, package-scoped structured logging), but telemetry/config handling introduces a competing confidentiality risk surface (`cli/pkg/tasker/errors.go::ErrTaskFailed`, `cli/internal/logger/log.go::New`, `cli/internal/cmd/telemetry/utils.go::TrackCommand`, `cli/internal/config/config.go::updateConfig`) — see `patterns.md` and `concerns.md`.
- Toolchain choices reinforce the architecture: Cobra/Viper-centered command bootstrapping and the documented orientation entrypoints align with the runtime composition model, while local `replace` usage highlights environment-coupling debt in the same integration seam (`go.mod::module github.com/rudderlabs/rudder-iac`, `go.mod::replace github.com/rudderlabs/rudder-data-catalog-provider/sdk => ../rudder-data-catalog-provider/sdk`, `cli/internal/cmd/root.go::Execute`) — see `stack.md` and `entry-points.md`.

## RUD-2752 — Event Stream Source Listing Layer Placement
<!-- ticket:RUD-2752 -->
- Workspace event-stream source listing is intentionally implemented at CLI/provider layers, not by changing control-plane or low-level API client behavior.
- The command entry delegates through provider `List` dispatch into source handler list logic, preserving existing list-command architecture and keeping blast radius limited.
- This layering relies on pre-existing paginated source retrieval in the event stream API client, so feature additions can be composed above client transport when read-path primitives already exist.

## INT-6489 — Destination API Versioning DTO Surface
<!-- ticket:INT-6489 -->
- The shared API client owns destination DTO shape and CRUD transport in `api/client/destinations.go`, so public API destination contract fields should be modeled there first.
- Destination versioning is represented on the public client DTO as `Destination.Version` and `Destination.VersionInfo`, allowing Create, Update, and Get paths to share one contract type.
- Optional destination version metadata flows through the existing shared service helper and response unmarshal path without separate service-method changes.

## INT-6671 — RETL Sync Behaviour Request Versus Response Contract
<!-- ticket:INT-6671 -->
- `CreateRETLConnectionRequest` in `api/client/retl/connection_types.go` owns the request-only `syncBehaviour` contract and models it as optional, so nil create requests omit the JSON key.
- `RETLConnection.SyncBehaviour` in the same file intentionally remains a non-pointer value field because create/list/get responses should continue exposing the resolved server mode.
- Keep RETL create-request DTO changes separate from RETL response DTO changes; request optionality should not erase the resolved sync mode returned by the API.

## RUD-2899 — DataGraph General Availability Wiring
<!-- ticket:RUD-2899 -->
- DataGraph is now a default project/provider capability rather than an experimental feature: dependency assembly should initialize `providers.DataGraph` and include `"datagraph"` in the composite provider map unconditionally alongside DataCatalog, RETL, EventStream, and Transformations.
- The `data-graphs` command is intended to be visible in the root Cobra command tree by default; command visibility should not depend on `ExperimentalFlags.DataGraph`.
- DataGraph GA means project-level validation and apply flows can encounter `kind: data-graph` / `version: rudder/v1` specs without opt-in, so shared project gatekeeper rule surfaces need to account for that match pattern.

## RUD-2860 — Destination External ID Mutation Boundary
<!-- ticket:RUD-2860 -->
- Destination external IDs are part of the shared API client destination DTO/read contract (`Destination.ExternalID` with `json:"externalId,omitempty"`), so create/read/list/get transport can carry the field through `api/client/destinations.go`.
- Ownership metadata mutation is intentionally isolated from ordinary destination update: destination external IDs are set through `Destinations.SetExternalID(ctx, id, externalID)`, which PUTs `{"externalId": externalID}` to `/v2/destinations/:id/external-id`.
- Destination update should not be treated as the external-ID ownership-metadata mutation path; update requests clear `ExternalID` before marshaling even when the caller's `Destination` struct has it populated.

## DEX-510 — HTTP Destination Registration Boundary
<!-- ticket:DEX-510 -->
- HTTP destination support is implemented as a destination-provider definition under `cli/internal/providers/destination/definitions/http`, with registry wiring in `cli/internal/app/dependencies.go`.
- HTTP is treated as an unverified destination: it should register only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled, matching the S3 destination gate.
- HTTP destination identity is `Type: http`, API type `HTTP`, and version `1`; allowed source types are intentionally restricted to the CLI-owned event-stream set (`android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `react_native`, `flutter`, `cordova`, `cloud`) while upstream `amp`, `warehouse`, and `shopify` are excluded.

## DEX-608 — S3 Destination Verified Registry Promotion
<!-- ticket:DEX-608 -->
- S3 is a verified/native destination definition: `cli/internal/app/newDestinationRegistry` should register `s3.NewDefinition()` whenever `ExperimentalFlags.DestinationSupport` is enabled, without requiring `ExperimentalFlags.UnverifiedDestinations`.
- HTTP remains an unverified destination definition and still requires both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` before `http.NewDefinition()` is registered.
- Destination registry flag-matrix expectations should reflect S3 as available with destination support alone, while enabling unverified destinations adds HTTP alongside S3.
- This supersedes earlier destination-registry guidance that used S3 as an unverified peer example for HTTP; HTTP remains gated, but S3 no longer shares that gate.

## DEX-499 — GCS Destination Unverified Registry Placement
<!-- ticket:DEX-499 -->
- Every newly onboarded destination definition registers as unverified: it goes inside the `ExperimentalFlags.UnverifiedDestinations` block in `cli/internal/app/dependencies.go`, requiring both `DestinationSupport` and `UnverifiedDestinations`.
- GCS is treated as an unverified destination definition on that rule, alongside Attentive Tag, Customer.io Audience, HTTP, and RS.
- S3 remains the verified/native cloud-storage destination registered with `ExperimentalFlags.DestinationSupport` alone; do not use S3's gate as the GCS registry precedent.
- Promotion to the verified block is a separate, deliberate step once a definition has been verified against a live stack — never part of the onboarding change itself.

## DEX-498 — Facebook Conversions Destination Onboarding
<!-- ticket:DEX-498 -->
- `facebook_conversions` is treated as an unverified destination definition for registry wiring: register it only when destination support and unverified destinations are enabled, not with verified/native S3-style destination support alone.
- `facebook_conversions` source-type support intentionally includes every db-config source type that has a destination common mapping: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.

## DEX-505 — Google Pub/Sub Destination Onboarding
<!-- ticket:DEX-505 -->
- `googlepubsub` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Google Pub/Sub uses CLI type `googlepubsub` for API type `GOOGLEPUBSUB`. Three names are in play — the integrations-config directory (`googlepubsub`), the terraform registration (`google_pubsub`), and the API type (`GOOGLEPUBSUB`) — and CLI resource identity resolves to `lowercase(APIType)`, keeping every definition's `Type` consistent with its `APIType`.
- Google Pub/Sub follows the cloud-storage-style source-type boundary used by GCS/S3: retain the CLI-owned event-stream source types (`android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `react_native`, `flutter`, `cordova`, `cloud`) with cloud connection mode, and exclude upstream `amp`, `warehouse`, and `shopify` until ownership/mapping is explicit.

## DEX-516 — Kinesis Destination Onboarding
<!-- ticket:DEX-516 -->
- `kinesis` is treated as an unverified destination definition: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Kinesis destination identity is CLI type `kinesis`, API type `KINESIS`, and version `1`.
- Kinesis follows the cloud-storage-style source-type boundary used by GCS/S3/Google Pub/Sub: retain the CLI-owned event-stream source types (`android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `react_native`, `flutter`, `cordova`, `cloud`) with cloud-only connection mode.
- Kinesis secrets are local YAML config keys `access_key_id` and `access_key`.
## DEX-521 — Marketo Destination Onboarding
<!-- ticket:DEX-521 -->
- `marketo` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Marketo source-type support intentionally includes every integrations-config/db-config supported source type that has a CLI local mapping: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.
- Marketo supports cloud connection mode for all retained source types.
- Marketo does not need destination-specific gated key paths when all mapped API keys are already present in db-config `defaultConfig`.
## DEX-523 — Salesforce Destination Onboarding
<!-- ticket:DEX-523 -->
- `salesforce` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Salesforce uses CLI type `salesforce`, API type `SALESFORCE`, and destination version `1`, even though upstream integrations metadata marks the classic Salesforce destination as deprecated in favor of Salesforce V2.
- Salesforce source-type support intentionally preserves every upstream/catalog and Terraform-supported source type that the common destination mapper supports: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`, all with cloud-only connection mode.

## DEX-525 — Slack Destination Onboarding
<!-- ticket:DEX-525 -->
- Slack destination support is implemented as a destination definition under `cli/internal/providers/destination/definitions/slack`.
- Slack local config intentionally includes only Terraform-mapped fields: `webhook_url`, `identify_template`, `event_channel_settings` (`name`, `channel`, `regex`), `event_template_settings` (`name`, `template`, `regex`), `whitelisted_trait_settings`, and `consent_management`.
- Upstream schema/db-config fields `incomingWebhooksType`, `eventChannelWebhook`, and `denyListOfEvents` are intentionally omitted from the CLI Slack config surface until they have Terraform/common mapping support; the destination onboarding workflow treats Terraform mappings as the property source of truth.

## DEX-677 — Confluent Cloud Destination Onboarding
<!-- ticket:DEX-677 -->
- `confluent_cloud` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Confluent Cloud destination support is implemented as a destination definition under `cli/internal/providers/destination/definitions/confluent_cloud`, with integrations-config left read-only for onboarding changes.
- Confluent Cloud models schema-only common consent category/purpose surfaces as destination-specific local config blocks `one_trust_cookie_categories` and `ketch_consent_purposes`, using mechanical camelCase-to-snake_case naming and direct API-key mappings so update does not erase UI-set values.

## DEX-515 — Postgres Destination Onboarding
<!-- ticket:DEX-515 -->
- Postgres destination source-type support intentionally follows the mapped db-config/common destination surface rather than Snowflake/S3-style narrowing.
- Postgres should include the local source types `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `react_native`, `cloud_source`, `flutter`, `cordova`, and `shopify` when no explicit unmapped registry/product constraint requires narrowing.

## DEX-493 — BigQuery Destination Onboarding
<!-- ticket:DEX-493 -->
- BigQuery destination support is implemented as CLI destination type `bq` for API type `BQ` under `cli/internal/providers/destination/definitions/bq`.
- BigQuery follows the broad warehouse source-type set from integrations-config `db-config.json`, not the narrowed cloud-storage/event-stream-owned source set: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `react_native`, `cloud_source`, `flutter`, `cordova`, and `shopify`, all cloud-only.
## DEX-520 — S3 Datalake Destination Onboarding
<!-- ticket:DEX-520 -->
- `s3_datalake` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- S3 Datalake destination identity is CLI type `s3_datalake`, API type `S3_DATALAKE`, destination version `1`, and definition package path `cli/internal/providers/destination/definitions/s3_datalake` using Go package name `s3datalake`.
- S3 Datalake config is intentionally modeled as flat local YAML, including auth/storage/sync/table-layout flags and consent management, so it maps directly to flat API keys and preserves schema/db-config keys that would otherwise be erased by whole-config destination updates.
- Model schema/db-config keys without Terraform mappings for S3 Datalake instead of omitting them: `skip_tracks_table`, `skip_users_table`, `time_window_layout`, `underscore_divide_numbers`, `cleanup_object_storage_files`, and `allow_users_context_traits`.

## DEX-690 — Redshift Destination Re-Onboarding
<!-- ticket:DEX-690 -->
- Redshift destination support is implemented as local type `rs`, API type `RS`, and destination version `1`; it remains an unverified destination registered only when both destination support and unverified destinations are enabled.
- Redshift re-onboarding should model the full current RS db-config/defaultConfig surface in `cli/internal/providers/destination/definitions/rs`, including IAM auth fields, SSH fields, flat sync fields, object-storage auth/prefix/cleanup fields, warehouse skip/prefer/json/immutable fields, and consent mapping.
- Redshift should not rely on framework-level unknown API config preservation: destination handler conversion only emits registered definition properties, so omitted Redshift config keys can be dropped during update/import.
## DEX-504 — Google Sheets Destination Onboarding
<!-- ticket:DEX-504 -->
- Google Sheets destination support is implemented as CLI destination type `googlesheets` for API type `GOOGLESHEETS` under `cli/internal/providers/destination/definitions/googlesheets`; Terraform's `google_sheets` registration name is not used for CLI resource identity.
- `googlesheets` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.

## DEX-509 — Kafka Destination Onboarding
<!-- ticket:DEX-509 -->
- `kafka` is treated as an unverified destination definition: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Kafka destination registry code order follows the explicit onboarding placement after Kinesis and before Marketo inside the unverified block, while exported/supported type lists may still be sorted lexicographically by implementation.

## DEX-487 — ActiveCampaign Destination Onboarding
<!-- ticket:DEX-487 -->
- ActiveCampaign destination support is implemented as CLI local type `active_campaign`, API type `ACTIVE_CAMPAIGN`, and destination version `1`.
- `active_campaign` is treated as an unverified destination definition: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- ActiveCampaign local config is intentionally flat: `api_url`, `api_key`, `actid`, `event_key`, plus shared `consent_management`; `connection_mode`, `use_native_sdk`, and legacy oneTrust/Ketch include-key blocks are not ordinary definition config fields.

## DEX-492 — Braze Destination Onboarding
<!-- ticket:DEX-492 -->
- `braze` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Braze destination support is implemented as a CLI destination definition under `cli/internal/providers/destination/definitions/braze`, with integrations-config left read-only for onboarding changes.
## DEX-494 — Customer.io Destination Onboarding
<!-- ticket:DEX-494 -->
- Customer.io destination support is implemented as CLI local type `customerio`, API type `CUSTOMERIO`, destination version `1`, and definition package path `cli/internal/providers/destination/definitions/customerio`.
- `customerio` is treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Customer.io source-type support intentionally keeps the broad mapped upstream set rather than the S3/GCS-style storage subset: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`.
- Customer.io local config maps `site_id`, `api_key`, `device_token_event_name`, `datacenter`, SDK/device nested source blocks, `event_filtering` whitelist/blacklist discriminator, and shared `consent_management`; `use_native_sdk` remains a source-type config block rather than an ordinary gated property.

## DEX-508 — Intercom Destination Onboarding
<!-- ticket:DEX-508 -->
- Intercom destination support is implemented as the legacy CLI destination type `intercom`, API type `INTERCOM`, and destination version `1`; do not substitute `intercom_v2` / `INTERCOM_V2` for this onboarding path.
- The `INTERCOM_V2` contract remains excluded because it has no Terraform destination registration/mapping source and represents a separate OAuth/account-management contract.
- `intercom` should be treated as an unverified destination definition for registry wiring: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Intercom's modeled config surface includes `app_id`, `api_key`, `api_server`, `api_version`, `send_anonymous_id`, `update_last_request_at`, source-scoped `connection_mode`, source-scoped `use_native_sdk`, Android-only `mobile_api_key_android`, iOS-only `mobile_api_key_ios`, `event_filtering`, and shared `consent_management`; `api_key` is the sole db-config secret key.
## DEX-497 — Facebook Pixel Destination Onboarding
<!-- ticket:DEX-497 -->
- Facebook Pixel destination support is implemented as CLI local type `facebook_pixel`, API type `FACEBOOK_PIXEL`, destination version `1`, and definition package path `cli/internal/providers/destination/definitions/facebook_pixel`.
- `facebook_pixel` is treated as an unverified destination definition: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Facebook Pixel uses broad Facebook source types: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`; `web` supports both cloud and device modes while every other source type is cloud-only.
- Facebook Pixel `SecretKeys` contains local `access_token`, following db-config `accessToken` secret metadata even though `schema.json` omits that field.

## DEX-518 — Qualtrics Destination Onboarding
<!-- ticket:DEX-518 -->
- Qualtrics destination support is implemented as CLI local type `qualtrics`, API type `QUALTRICS`, destination version `1`, and definition package path `cli/internal/providers/destination/definitions/qualtrics`.
- `qualtrics` is treated as an unverified destination definition: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Qualtrics source-type support is intentionally narrow: only `web`, `android`, and `ios` are supported, all in device-only connection mode.

## DEX-719 — GCS Connection Mode Re-Onboarding
<!-- ticket:DEX-719 -->
- GCS is an existing destination definition, so connection-mode re-onboarding should update the existing GCS definition/tests/fixture-snapshot pairs rather than adding duplicate destination registry imports, app flag-matrix cases, or new definitions.
- GCS remains an unverified destination in `cli/internal/app/dependencies.go`; keep registry gating under both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations`.
- Integrations-config and Terraform remain read-only source-of-truth references for this re-onboarding path; the durable CLI-owned changes belong in `cli/internal/providers/destination/definitions/gcs` and destination E2E testdata.
## DEX-512 — LinkedIn Ads Destination Onboarding
<!-- ticket:DEX-512 -->
- `linkedin_ads` follows ads-destination source-type precedent, closer to Facebook Conversions than cloud-storage destinations: include every upstream db-config source type with a destination common mapping (`android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`) instead of trimming to the CLI-owned event-stream subset.
- LinkedIn Ads supported source connection modes are cloud-only for all retained source types, matching the linkedIn_ads db-config contract.
## DEX-527 — Snowpipe Streaming Models Its Whole Config Surface
<!-- ticket:DEX-527 -->
- Snowpipe Streaming models all 14 `destConfig.defaultConfig` keys, so nothing it owns is dropped when destination update replaces the whole config object; it needs no destination-specific preservation of unmodelled API keys.
- The keys that are genuinely unmodelled by every destination are the legacy consent blocks `oneTrustCookieCategories` and `ketchConsentPurposes`: `common.Properties` maps consent management but not those two, so any destination carrying them upstream loses them on update. That is a cross-destination gap, not a Snowpipe one, and belongs in its own change rather than an opt-in flag on a single definition.

## DEX-490 — Amplitude Destination Onboarding
<!-- ticket:DEX-490 -->
- Amplitude destination support is implemented as CLI destination type `am`, API type `AM`, destination version `1`, and definition package path `cli/internal/providers/destination/definitions/am`.
- `am` is treated as an unverified destination definition: register it only when both `ExperimentalFlags.DestinationSupport` and `ExperimentalFlags.UnverifiedDestinations` are enabled.
- Amplitude retains the broad mapped analytics source set (`android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `amp`, `cloud`, `warehouse`, `react_native`, `flutter`, `cordova`, and `shopify`) rather than the narrowed event-stream-owned set used by storage-like destinations.
- Amplitude `SecretKeys` contains only local `api_secret`, following db-config `apiSecret` secret metadata.

## DEX-731 — Concurrent Syncs GA
<!-- ticket:DEX-731 -->
- Apply and destroy project syncers should always be constructed with `syncer.WithConcurrency(config.GetConfig().Concurrency.Syncer)`; concurrent sync execution is no longer gated by an experimental flag.
- `concurrency.syncer` remains the main configuration tuning knob for sync parallelism after GA promotion; do not replace it with hard-coded concurrency or reintroduce a `concurrentSyncs` gate.

## DEX-732 — Destination Support General Availability Wiring
<!-- ticket:DEX-732 -->
- Destination support is now a default project/provider capability rather than an experimental feature: `cli/internal/config/experimental.go` should not define `DestinationSupport` / `destinationSupport`.
- Dependency assembly in `cli/internal/app/dependencies.go` should always construct the destination provider and include it in the composite provider map, without a `DestinationSupport` guard.
- `newDestinationRegistry` should always register verified/native S3; `ExperimentalFlags.UnverifiedDestinations` remains the gate for unverified destination definitions.
- This supersedes earlier destination-registry guidance that required both `DestinationSupport` and `UnverifiedDestinations` for unverified definitions: only `UnverifiedDestinations` remains as the experimental gate after destination support GA.
