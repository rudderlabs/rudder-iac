---
name: onboard-cli-destination
description: Onboard a new destination type into the rudder-iac CLI destination registry. Use when asked to add/onboard/register a destination (e.g. webhook, ga4, digital_ocean_spaces) in the CLI. Takes the destination's localType (its directory name under rudder-integrations-config destinations) as input, derives the upstream APIType from that directory's db-config.json, ports property mappings from terraform-provider-rudderstack, derives validations from rudder-integrations-config schema.json, and produces definition.go, definition_test.go, registry wiring, destination e2e fixtures/snapshots, and a valid example YAML.
---

# Onboard CLI Destination

Add a new destination type to the rudder-iac CLI destination registry
(`cli/internal/providers/destination/definitions/`).

**Input**: the destination's `localType` — its directory name under
`rudder-integrations-config/src/configurations/destinations/<name>` (e.g.
`s3`, `webhook`, `digital_ocean_spaces`). The upstream `APIType` is derived
automatically from that directory's `db-config.json` — do not ask the user
for it.

**Outputs**:

1. `cli/internal/providers/destination/definitions/<type>/definition.go`
2. `cli/internal/providers/destination/definitions/<type>/definition_test.go`
3. Registration in `cli/internal/app/dependencies.go` (`newDestinationRegistry`)
4. Destination e2e fixtures and expected snapshots for each meaningful config
   variation, or a documented deferral reason when a live snapshot cannot be
   captured safely
5. A valid example YAML spec, printed in the final response (not committed as a file)

Out of scope: rule-doc updates.

## Source-of-truth split (fixed roles — do not deviate)

| Concern | Source | Location |
| --- | --- | --- |
| Property mappings (local ↔ API config keys, reshapes) | terraform-provider-rudderstack | `../terraform-provider-rudderstack/rudderstack/integrations/destinations/destination_<name>.go` |
| Config validations (required, patterns, enums, conditionals) | integrations-config `schema.json` | `../rudder-integrations-config/src/configurations/destinations/<dir>/schema.json` |
| Source types, connection modes, secrets | integrations-config `db-config.json` | `../rudder-integrations-config/src/configurations/destinations/<dir>/db-config.json` |
| Structural template | S3 definition in this repo | `cli/internal/providers/destination/definitions/s3/definition.go` |

`<dir>` in the table above is the input `localType`.

Both sibling repos are checked out next to rudder-iac under
`~/workspace/go/src/github.com/rudderlabs/`.

Ignore `.specs/destination_definition_registry.md` — it describes a JSON-Schema
approach the code did not follow. The S3 definition is the source of truth.

## Workflow

### Step 1: Resolve APIType from localType

Read
`rudder-integrations-config/src/configurations/destinations/<localType>/db-config.json`
and take its `name` field — that is the upstream `APIType`.

- `s3` → `db-config.json` `name: "S3"` → `APIType = S3`
- `webhook` → `name: "WEBHOOK"` → `APIType = WEBHOOK`
- `digital_ocean_spaces` → `name: "DIGITAL_OCEAN_SPACES"` → `APIType = DIGITAL_OCEAN_SPACES`

If the directory or `db-config.json` doesn't exist, stop and tell the user —
do not guess a directory name.

Local CLI `Type` = `lowercase(APIType)`. This must equal the input
`localType` — if it doesn't (directory name and registered `name` disagree,
which happens for a handful of destinations), flag the mismatch in the final
report and use `lowercase(APIType)` as the canonical `Type`, not the raw
input.

Directory: `definitions/<type>/`. Go package name: type with underscores
stripped (`digital_ocean_spaces` dir → `package digitaloceanspaces`),
following the S3 precedent (`definitions/s3/`, `package s3`).

### Step 2: Locate the terraform source

The integrations-config directory is already known (`localType`, the input)
— no need to search for it.

- Terraform file: grep `APIType: "<APITYPE>"` under
  `terraform-provider-rudderstack/rudderstack/integrations/destinations/`.

If the destination is missing from terraform, stop and tell the user — the
skill requires terraform as mapping source (per team decision). Do not derive
mappings from ui-config on your own.

Read [reference/source-extraction.md](reference/source-extraction.md) for what
to extract from each file.

### Step 3: Refuse duplicates

Check `newDestinationRegistry` in `cli/internal/app/dependencies.go`. If the
`(APIType, Version)` pair is already registered, stop and report.

### Step 4: Write definition.go

Use the S3 definition as structural template. Read
[reference/definition-anatomy.md](reference/definition-anatomy.md) for the
annotated walkthrough, and
[reference/converter-mapping.md](reference/converter-mapping.md) for porting
terraform `ConfigProperty` calls to the CLI converter helpers.

Mechanical rules:

- API camelCase keys → snake_case local keys (`bucketName` → `bucket_name`).
- Port `converter.Simple` mappings **bare** — never pass `SkipZeroValue` or
  other value filters, even where terraform does. Enforce presence via
  validation (`required` / `required_if`), not by skipping empty values (see
  converter-mapping.md "Do not skip zero values").
- `Version` = terraform's registered `Version` (currently 1 everywhere).
- `schema.json` constraints → `validate` struct tags (see source-extraction.md
  for the constraint translation table). Real regex patterns →
  `pattern=<name>`: reuse an existing `NewPattern` registration, or register
  a minimal new one. Strip upstream `env.` / `{{ … }}` alternations; never
  bake those into the CLI regex (see source-extraction.md "Enforcing regex
  patterns").
- `db-config.json` `secretKeys` → `SecretKeys` field, translated to snake_case
  local keys.
- `db-config.json` `supportedSourceTypes` / `supportedConnectionModes` →
  `SourceTypes` / `ConnectionModes`, translated to CLI-local source types via
  [reference/source-type-mapping.md](reference/source-type-mapping.md).
  Unmapped upstream types (e.g. `amp`, `shopify`, `warehouse` when not
  CLI-owned): drop and flag in the final report — never guess.
- `db-config.json` `supportedSourcesValidation` (when present and non-empty) →
  `SupportedSourcesValidation`: keys translated to CLI-local source types via
  the same mapping, values translated from API camelCase field names to
  snake_case local keys. Drop entries whose source type was dropped from
  `SourceTypes` and flag them in the final report. Most destinations have no
  `supportedSourcesValidation` — omit the field then; never invent entries.
- Source-type-gated keys: if a terraform-mapped property's API key is absent
  from `db-config.json` `destConfig.defaultConfig` but present under specific
  `destConfig.<sourceType>` lists, wrap the ported property in
  `converter.Gated(prop, localSourceTypes...)` with the local source types
  that list it (see source-extraction.md "Detecting gated keys" for the scan
  procedure and boilerplate keys to skip). The registry indexes these at
  Register() and exposes them via `GatedKeyPaths()`.
- If the destination supports consent management (nearly all do): append
  `common.Properties(sourceTypes)...` to properties and add a
  `ConsentManagement common.ConsentManagement` field tagged
  `mapstructure:"consent_management"` to the config struct. The registry
  rejects any other type for that field.

### Step 5: Register in dependencies.go

Add the import and one line inside `newDestinationRegistry`:

```go
if err := registry.Register(<pkg>.NewDefinition()); err != nil {
    return nil, fmt.Errorf("registering <type> destination definition: %w", err)
}
```

Registration itself validates the definition (source types mapped, connection
modes complete for every source type, consent field type). A broken definition
fails every test that builds the registry — good early signal.

### Step 6: Write definition_test.go

Mirror `definitions/s3/definition_test.go` exactly in structure:

1. `TestNewDefinitionMetadata` — register in fresh registry, assert `Type`,
   `APIType`, `Version`, `SecretKeys()`, `SupportedSourceTypes()`,
   `ConnectionModes()` per source type, and `GetByAPIType` lookup. When the
   definition carries `SupportedSourcesValidation`, also assert
   `SupportedSourcesValidation(sourceType)` per configured source type and
   `Nil` for one source type without an entry. When the
   definition has gated properties, also assert the full `GatedKeyPaths()`
   map with `assert.Equal` (JSON-pointer keypaths, e.g.
   `map[string][]string{"/mobile_api_key_android": {"android"}}`).
2. `Test<Type>ConfigValidation` — subtests via `registered.ValidateConfig`:
   each required field missing, each conditional/exclusion rule, each
   `pattern=` field with an invalid literal rejected, valid minimal config,
   valid full config, unknown key rejected, and (when consent is supported)
   unsupported consent source + invalid provider.
3. `Test<Type>ConversionRoundTrip` — `testutil.AssertConversion` with paired
   `LocalJSON`/`APIJSON` cases: minimal, full, each reshape (arrays,
   discriminators), consent for at least one boundary-mapped source type
   (`android_kotlin` ↔ `androidKotlin`).

Testify only. `t.Parallel()` on tests and subtests, matching S3.

### Step 7: Generate and verify the example YAML

Write a spec with realistic values that satisfies every `validate` tag:

```yaml
version: rudder/v1
kind: destination
metadata:
  name: my-<type>-destination
spec:
  id: my-<type>-destination
  display_name: My <Display Name> Destination
  type: <type>
  enabled: true
  definition_version: <Version>
  config:
    # snake_case keys with realistic values
```

Verify it mechanically, not by eye: either include a `ValidateConfig` test
case using the exact example config, or run a temp-project
`rudder-cli project validate` with the experimental flag on. The example must
pass before it ships in the response.

### Step 8: Add destination e2e fixtures

Read [reference/e2e-tests.md](reference/e2e-tests.md). Add catalog-layout
fixtures for each meaningful config variation the destination supports, plus
matching expected upstream snapshots when a live destination-enabled stack is
available.

E2E complements, not duplicates, `definition_test.go`: unit tests remain
exhaustive for config validation and conversion surfaces, while
`TestDestinationsApply` covers the live apply → update → re-apply lifecycle for
meaningful backend-facing variations.

### Step 9: Verify

```bash
go build ./...
go test ./cli/internal/providers/destination/... ./cli/internal/app/...
go test ./cli/tests -run TestDestinationsApply -count=1
go test ./cli/tests/helpers -run TestDestinationSnapshotTester -count=1
make lint
```

All must pass. `cli/internal/app` is included because `dependencies_test.go`
and `ruledoc_test.go` exercise the registry with the new definition. The
ungated `TestDestinationsApply` run must compile and skip cleanly; run the gated
live e2e with `RUN_DESTINATION_E2E=1` when the required live stack is available.

### Step 10: Report

Final response must include:

- Files created/modified
- The validated example YAML
- Destination e2e coverage added: each meaningful variation covered by fixtures
  and snapshots, or the documented deferral reason
- E2E verification status: skip/compile result and whether the gated live run was
  performed
- Flagged discrepancies (terraform vs schema.json disagreements, dropped
  source types, dropped `supportedSourcesValidation` entries, upstream fields
  intentionally omitted)
- Gated keys: which properties were gated and to which source types; gates
  narrowed or properties omitted because their source types were dropped
- Reminder: usage requires `experimental: true` + `flags.destinationSupport: true`
  in the CLI config

## Guardrails

- Never register two definitions with the same `(Type, Version)` or
  `(APIType, Version)` — the registry errors, but check before writing code.
- When terraform mappings and `schema.json` disagree (field present in one,
  absent in the other; different optionality), follow terraform for the
  mapping and schema.json for the validation, and flag the discrepancy in the
  report. Do not silently invent a resolution.
- Do not add fields that exist upstream but have no terraform mapping — they
  are out of the supported surface. List them as omitted in the report.
- Terraform's `Negated` helper has no CLI converter equivalent; see
  converter-mapping.md before hand-rolling one.
- Do not leave real `schema.json` / terraform regex constraints unenforced
  when a named `pattern=` can express them. Do not create dynamic-supporting
  regexes (`env.` / `{{ … }}` branches) for destination config fields.
