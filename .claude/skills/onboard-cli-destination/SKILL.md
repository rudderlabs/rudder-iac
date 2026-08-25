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
3. Registration in `cli/internal/app/dependencies.go` (`newDestinationRegistry`),
   always inside the `UnverifiedDestinations` block, plus the matching
   `dependencies_test.go` flag-matrix expectation
4. Destination e2e fixtures and expected snapshots for each meaningful config
   variation, or a documented deferral reason when a live snapshot cannot be
   captured safely
5. A valid example YAML spec, printed in the final response (not committed as a file)

Out of scope: rule-doc updates.

## Source-of-truth split (fixed roles — do not deviate)

| Concern | Source | Location |
| --- | --- | --- |
| Property mappings (local ↔ API config keys, reshapes) | terraform-provider-rudderstack | `../terraform-provider-rudderstack/rudderstack/integrations/destinations/destination_<name>.go` |
| Config surface (which keys exist) and validations (required, patterns, enums, conditionals) | integrations-config `schema.json` | `../rudder-integrations-config/src/configurations/destinations/<dir>/schema.json` |
| Source types, connection modes, secrets, field allowlist | integrations-config `db-config.json` | `../rudder-integrations-config/src/configurations/destinations/<dir>/db-config.json` |
| Intent of a key terraform does not map (grouping, labels) | integrations-config `ui-config.json` | `../rudder-integrations-config/src/configurations/destinations/<dir>/ui-config.json` |
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

If the destination is missing from terraform **entirely**, stop and tell the
user — the skill requires terraform as mapping source (per team decision).

That is not the same as terraform missing individual keys, which is common.
Port everything terraform does map, then reconcile against `db-config.json`
`defaultConfig`: any key listed there (or in `schema.json`) with no terraform
mapping is filled in from integrations-config rather than dropped —

- **local key**: the API key, camelCase → snake_case (`useSASTokens` → `use_sas_tokens`);
- **shape**: flat unless `schema.json` declares the key as an object;
- **validation**: the `schema.json` pattern/enum/default for that key;
- **`ui-config.json`**: consult it to disambiguate intent — how the key is
  grouped, labelled, and which other keys it is shown with. Use it to interpret
  an unclear key, not as a mapping source of its own.

List every such key in the final report.

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
  `dynamic_or_pattern=<name>`: reuse an existing `NewPattern` registration, or
  register a minimal new one. Strip upstream `env.` / `{{ … }}` alternations;
  never bake those into the CLI regex — the tag accepts `{{ path || fallback }}`
  templates, the pattern holds only the real constraint. Note `^(.{0,100})$` is
  a pattern, not a length limit (it forbids newlines) — reuse the shared
  `single_line_100` rather than `max=100`. Use plain `pattern=<name>` only to
  reject templates too (see source-extraction.md "Enforcing regex patterns").
- `db-config.json` `secretKeys` → `SecretKeys` field, translated to snake_case
  local keys.
- `db-config.json` `supportedSourceTypes` / `supportedConnectionModes` →
  `SourceTypes` / `ConnectionModes`, translated to CLI-local source types via
  [reference/source-type-mapping.md](reference/source-type-mapping.md).
  Unmapped upstream types (e.g. `amp`, `shopify`, `warehouse` when not
  CLI-owned): drop and flag in the final report — never guess.
- `schema.json` `configSchema.allOf` branches conditioned on `connectionMode` →
  `SupportedSourcesValidation`, a
  `map[localSourceType]map[connectionMode][]localConfigKey`: only keys the
  branch's `then.required` makes **required**, never optional ones. Source types
  translated to CLI-local types via the same mapping, keys to snake_case.
  **There is no `supportedSourcesValidation` key in db-config** — do not look
  for one. A branch whose requiredness also depends on another config value
  (Braze `usePlatformSpecificApiKeys`) is not expressible: omit it and flag it.
  See source-type-mapping.md "Per-source-type connect-time required keys" for
  the recognised `if` shapes and the full derivation; omit the field when no
  kept source type contributes a key.
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

**Every newly onboarded destination registers as unverified.** Add the import
and one block inside `newDestinationRegistry`, within the
`cfg.ExperimentalFlags.UnverifiedDestinations` branch — never the verified
section above it:

```go
if cfg.ExperimentalFlags.UnverifiedDestinations {
    // ... existing unverified registrations, kept alphabetical by type ...
    if err := registry.Register(<pkg>.NewDefinition()); err != nil {
        return nil, fmt.Errorf("registering <type> destination definition: %w", err)
    }
}
```

The verified section (registered on `DestinationSupport` alone) is reserved for
definitions already proven against a live stack — S3 today. Promotion into it is
a separate, deliberate change after that verification; it is never part of the
onboarding PR, no matter how simple the destination looks.

Then update the flag-matrix expectation in
`cli/internal/app/dependencies_test.go`: add the new type to the
`wantTypes` list of the both-flags-enabled case only (`SupportedTypes()` is
sorted, so insert alphabetically). The verified-only case must stay unchanged —
if adding the type there makes a test pass, the registration landed in the wrong
block.

Registration itself validates the definition (source types mapped, connection
modes complete for every source type, consent field type). A broken definition
fails every test that builds the registry — good early signal.

### Step 6: Write definition_test.go

Mirror `definitions/s3/definition_test.go` exactly in structure:

1. `TestNewDefinitionMetadata` — register in fresh registry, assert `Type`,
   `APIType`, `Version`, `SecretKeys()`, `SupportedSourceTypes()`,
   `ConnectionModes()` per source type, and `GetByAPIType` lookup. When the
   definition carries `SupportedSourcesValidation`, also assert
   `SupportedSourcesValidation(sourceType, connectionMode)` per configured
   pair and `Nil` for one supported pair without an entry. When the
   definition has gated properties, also assert the full `GatedKeyPaths()`
   map with `assert.Equal` (JSON-pointer keypaths, e.g.
   `map[string][]string{"/mobile_api_key_android": {"android"}}`).
2. `Test<Type>ConfigValidation` — subtests via `registered.ValidateConfig`:
   each required field missing, each conditional/exclusion rule, each
   pattern-validated field with an invalid literal rejected and a
   `{{ path || fallback }}` template accepted, valid minimal config,
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
  source types, connectionMode-conditioned required keys that could not be
  expressed in `SupportedSourcesValidation`, upstream fields
  intentionally omitted, and every `schema.json` key modelled without a
  terraform mapping — name each one and the local key you derived for it)
- Gated keys: which properties were gated and to which source types; gates
  narrowed or properties omitted because their source types were dropped
- Reminder: usage requires `experimental: true` + `flags.destinationSupport: true`
  + `flags.unverifiedDestinations: true` in the CLI config — newly onboarded
  destinations are always behind the unverified gate (Step 5)

## Guardrails

- Never register two definitions with the same `(Type, Version)` or
  `(APIType, Version)` — the registry errors, but check before writing code.
- **`schema.json` is the source of truth for the config surface**, with
  `db-config.json` for capabilities and secrets and `ui-config.json` for intent.
  Terraform supplies the mapping shape (reshapes, nested key names), but it
  bounds neither which keys exist nor how they are shaped. When the two
  disagree on optionality or constraints, follow schema.json and flag it.
- **Port every terraform mapping, then fill the gaps from db-config.** The two
  are additive: terraform's mappings stand as-is, and keys it does not map are
  derived from `db-config.json` `defaultConfig` plus `schema.json` (Step 2)
  rather than dropped. A key missing from terraform is an omission there, not
  evidence the key is unsupported. Secrets work the same way: take the whole of
  `db-config` `secretKeys`, using terraform's `Sensitive: true` only as a
  cross-check, since it routinely misses entries.
- **Terraform's shape only holds where it matches the upstream payload.**
  The upstream config is flat unless `schema.json` declares an object, so a flat
  local model is the default. Terraform's TF-list nesting (`s3.0.access_key_id`)
  is a provider artefact — mirroring it produces a model that cannot represent
  real payloads. Snowflake is the worked example: it sends `cloudProvider` and
  `roleBasedAuth` as first-class keys and carries every provider's keys at once
  (an `AZURE` destination still sends `bucketName`), so terraform-shaped
  `s3`/`gcp`/`azure` blocks plus a `Conditional` on `cloudProvider` silently
  dropped keys. Nest only where upstream genuinely nests (`excludeWindow`).
- **A key in `schema.json` with no terraform mapping must still be modelled.**
  Derive the local key mechanically (camelCase → snake_case) and validate it
  from schema.json. Omitting it does not make it "unsupported" — destination
  update replaces the whole config object, so any key the definition does not
  model is dropped from the payload and **erased upstream on the first apply**,
  including values a user set in the UI. This applies to nested keys inside
  array reshapes too.
  Slack is the worked example: `incomingWebhooksType`, `denyListOfEvents`, and
  the nested `eventChannelWebhook` are absent from terraform, and leaving them
  out made the backend drop `incomingWebhooksType` between create and update —
  visible as a create/update snapshot mismatch in the gated e2e.
- Terraform's `Negated` helper has no CLI converter equivalent; see
  converter-mapping.md before hand-rolling one.
- Do not leave real `schema.json` / terraform regex constraints unenforced
  when a named pattern can express them — including `^(.{0,100})$`-style
  constraints, which are patterns rather than length limits. Template support
  belongs in the tag (`dynamic_or_pattern=<name>`), never in the regex: do not
  register patterns carrying `env.` / `{{ … }}` branches, and never give the
  deprecated `env.VAR` form an escape hatch.
