# DEX-1000 carry-over audit: clean-room vs provider-owned schema generation

This is the merge-gate audit for the JSON Schema work. It compares the
clean-room `origin/feat/cli-json-schema` implementation (#649), the intended
provider-owned architecture described for #643, and this implementation.

| Capability / artifact | Source | Status | Resolution |
| --- | --- | --- | --- |
| Draft 2020-12 schemas reflected from typed spec structs | #649 | Carried over | `cli/internal/schema` reflects provider-supplied body types. |
| Full `version` / `kind` / `metadata` / `spec` envelope | #649 | Carried over and extended | Per-kind schemas pin `kind`, constrain every supported version, and require the reflected common metadata shape. |
| Validation-tag enrichment: `required`, `oneof`, `eq`, bounds | #649 | Carried over and extended | Bounds use string, collection, or numeric keywords according to Go type. |
| Nested `$defs` / `$ref` recursion safety | #649 | Carried over | Enrichment tracks schema/type pairs and root composition namespaces definitions. |
| Hardcoded kind-to-type registry | #649 | Deliberately dropped | Provider-owned `SpecSchemas()` avoids a second kind registry. |
| Drift test proving reflected struct fields reach output | #649 | Carried over | `cli/internal/schema/drift_test.go`. |
| Draft 2020-12 compile and valid/invalid acceptance tests | #649 | Carried over and broadened | Schema tests reject missing/invalid fields; app acceptance compiles every active schema/root and validates full E2E spec fixtures. |
| `SchemaProvider.SpecSchemas()` next to provider ownership | #643 design | Carried over | Optional interface in `cli/internal/provider`; each active project provider declares its schemas. |
| Composite aggregation with duplicate protection | #643 design | Carried over | `CompositeProvider.SpecSchemas()` collects schemas from the same provider set whose unique kind ownership is enforced during composite construction; provider and app tests assert schema/kind equality. |
| Active schema kinds equal `SupportedKinds()` | #643 design / issue | Carried over | Tests cover defaults, DataGraph, and flag-gated RETL table/connection kinds. |
| DataGraph schema | Issue requirement | Carried over | DataGraph is included by default and inline models/relationships are reflected. |
| Legacy/v1 bodies sharing one kind | Current-main requirement | Carried over | `properties`, `events`, and `custom-types` select their legacy or v1 body shape by `version`; structurally identical category bodies share one schema. |
| `rudder-cli schema` list and single-kind stdout | #649 | Carried over | Kind listing is deterministic and generation requires no credentials. |
| `--out` per-kind artifacts and discriminated root | #649 | Carried over | Writes `<kind>.schema.json` and `rudder-cli.schema.json`. |
| Safe output overwrite behavior | Issue requirement | Carried over and extended | Existing artifacts fail atomically unless `--overwrite` is explicit. |
| Writer/import modeline | #649 | Carried over with safer default | Opt-in `--schema-modeline`; default URL targets the running version's release assets. |
| Custom modeline URL root | Issue requirement | Carried over | `--schema-url-base` implies modelines and appends the CLI version exactly once. |
| Migration modeline path | Issue requirement | Carried over | Hidden migration command exposes the same controls and writer option. |
| Default relative `.rudder/schemas` modelines | #649 | Deliberately dropped | Relative paths were fragile for nested imported files; versioned release URLs are stable across output directories. |
| Automatic/default-on modelines | #649 | Deliberately dropped | Users opt in explicitly; GoReleaser publishes the versioned schema assets used when enabled. |
| VS Code documentation | #649 | Carried over | `docs/json-schema.md` documents modelines and `yaml.schemas`. |
| JetBrains documentation | Issue requirement | Carried over | `docs/json-schema.md` documents local schema mappings. |
| SchemaStore publication | Issue context | Deliberately deferred | No stable SchemaStore URL exists; release asset convention is documented instead. |
| New dependency justification | Issue requirement | Carried over | `invopop/jsonschema` generates schemas; `santhosh-tekuri/jsonschema/v6` compiles Draft 2020-12 output in tests. |

No comparison row remains missing. Closing or commenting on GitHub issues/PRs is
an external project-lead action and is intentionally not performed by this
repository change.
