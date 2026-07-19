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

## DEX-543 — Destination Secret Path Helper Boundary
<!-- ticket:DEX-543 -->
- Nested destination secret support is localized to the destination provider helper layer: `configpath` performs manual `map[string]any` path walks, while `SecretKeys` remains the definition-facing `[]string` API.
- The destination converter layer can already address dotted local keys separately, so dotted `SecretKeys` should be handled by wrap/mask/reveal/unknown helpers without changing definition shape or flattening nested YAML.
- Revealing destination secrets is part of API payload preparation, but the source config is live resource state; reveal must isolate writes with path clone-on-write rather than mutating the caller map or relying on a root-only shallow clone for nested paths.
