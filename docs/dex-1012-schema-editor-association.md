# DEX-1012: stable schema URLs and editor association

**Status:** spike decision, 2026-09-25

**Scope:** decision and editor proof only. **Part 2 is deferred and not
implemented by this spike.** Publication, importer/scaffolding changes,
SchemaStore submission, and Hugo narrative docs remain blocked on DEX-1000 and
DEX-1004.

## Decision

1. In Part 2, publish a kind-discriminated root and each per-kind schema under
   a Rudder-owned, spec-versioned URL. The proposed future canonical URLs (not
   published by this spike) are
   `https://www.rudderstack.com/docs/schemas/rudder-cli/v1/root.json` and
   `.../v1/<kind>.json`.
2. Make the root URL the default modeline written by import/scaffolding. A file's
   `kind` selects its `spec` schema. Generated/exported files may use a per-kind
   URL when the kind is already known.
3. Use `# yaml-language-server: $schema=<URL>` as the primary association for
   arbitrary filenames. Document VS Code `yaml.schemas`, JetBrains JSON Schema
   mappings, and future rudder-next-vscode content detection as bulk/automatic
   alternatives.
4. A SchemaStore entry is worth proposing only as additive discovery for an
   opt-in `*.rudder.yaml` / `*.rudder.yml` convention. It must not be presented
   as coverage for existing arbitrary names.
5. Attach the same generated schema bundle to CLI releases for pinned/offline
   use, but do not use a moving release URL as the canonical modeline target.

The reproducible two-kind demo and headless language-server evidence are in
[`dex-1012-demo/`](dex-1012-demo/README.md). The VS Code UI screenshot/GIF
acceptance proof is explicitly still incomplete because this spike environment
could not run VS Code with the Red Hat YAML extension.

## 1. SchemaStore matching

SchemaStore catalogs associate schemas by path globs; they do not inspect YAML
content. Current comparable entries illustrate the boundary:

| Tool | Current catalog association | How arbitrary names are handled |
| --- | --- | --- |
| GitHub Workflow | `**/.github/workflows/*.yml` and `*.yaml` (also Gitea/Forgejo workflow directories) | Directory convention makes any basename safe. |
| Docker Compose | `**/docker-compose.yml`, `compose.yml`, and `docker-compose.*` / `compose.*` variants | Narrow naming convention, not content detection. |
| Kubernetes | No generic Kubernetes-manifest SchemaStore catalog entry; `kubernetes-definitions.json` is a definitions document, not a matched catalog schema | Kubernetes-specific editor extensions/settings provide schema selection; arbitrary manifest names are not solved by SchemaStore. |

Sources: SchemaStore's current
[`catalog.json`](https://github.com/SchemaStore/schemastore/blob/master/src/api/json/catalog.json)
and [`kubernetes-definitions.json`](https://github.com/SchemaStore/schemastore/blob/master/src/schemas/json/kubernetes-definitions.json)
(accessed 2026-09-25).

### Option assessment

| Option | Coverage | Cost / limitation | Recommendation |
| --- | --- | --- | --- |
| `*.rudder.yaml` convention + SchemaStore | Zero-config for newly named files in editors using SchemaStore | Misses existing `events.yaml`, `sources/*.yaml`, etc.; renaming is disruptive | Optional additive convention and PR |
| Modeline | Exact, portable association for any filename; importer already plans to emit it | One comment per file; remote use needs network/cache | **Primary** |
| Checked-in `.vscode/settings.json` | Convenient bulk mapping per repository | VS Code-specific; generated settings can overwrite/team-conflict and globs may still be broad | Document; init may offer opt-in generation, not silently write it |
| rudder-next-vscode content detection | Best zero-comment experience; can inspect `version` + `kind` | VS Code-only and depends on RUD-2357 | Preferred automatic enhancement, not portable baseline |

A broad SchemaStore match such as `**/*.yaml` is unsafe because it would attach
Rudder validation to unrelated YAML. A narrow entry is low-risk and improves
discovery for adopters, so open it after the public root URL exists, with a
clear description that arbitrary names require another association.

## 2. Stable URL

Use a docs-hosted URL, versioned by the **spec contract** (`rudder/v1`), not by
the CLI binary release. These are proposed future canonical URLs; this spike
does not publish them:

```text
https://www.rudderstack.com/docs/schemas/rudder-cli/v1/root.json
https://www.rudderstack.com/docs/schemas/rudder-cli/v1/event-stream-source.json
https://www.rudderstack.com/docs/schemas/rudder-cli/v1/data-graph.json
```

| Candidate | Strengths | Weaknesses |
| --- | --- | --- |
| Docs-hosted | Short, product-owned URL; Hugo can publish the DEX-1004 versioned bundle; old spec versions can remain immutable; control over `application/schema+json`, CORS, cache policy, redirects | Requires docs deployment/CDN contract and monitoring |
| GitHub release asset | Immutable per CLI release; downloadable bundle is good offline/pinned input | CLI versions do not define the schema contract; asset URLs are long; `latest` redirects make caching/reproducibility less clear; cross-origin behavior is GitHub-owned |
| Raw `schemas` branch | Simple initial publishing | Branch contents are mutable, URL exposes repository implementation, MIME/cache/CORS are GitHub-owned, and branch lifecycle becomes a public contract |

Publication requirements for Part 2:

- Never mutate an existing `/v1/` schema incompatibly. Add `/v2/` when the spec
  version changes; compatible fixes can republish with CDN revalidation.
- Return a JSON/schema content type, allow editor GETs cross-origin, use ETags,
  and choose a bounded cache lifetime so compatible corrections propagate.
- Add a release asset containing the exact versioned bundle for offline use.
  Modelines and editor mappings accept `file:` or relative paths, so users can
  point at the downloaded bundle without changing the public URL contract.
- Test direct and referenced-schema URLs through the production CDN. Do not
  require a redirect to resolve routine `$ref`s.

## 3. Root versus per-kind schemas

Publish both; use the root by default. A single root gives importer/init one URL,
keeps association valid if a user changes `kind`, and permits top-level kind
completion. The root should use Draft 7-compatible `if`/`then` branches and each
`if` must include `required: [kind]`: `properties.kind.const` alone also passes
when `kind` is absent.

The demo tested `event-stream-source` (simple resource) and `data-graph` (nested
resource) with yaml-language-server 1.24.0:

| Exercised state | Root | Event-stream-source per-kind |
| --- | --- | --- |
| Blank document | Envelope completion | Envelope completion |
| Recognized `kind` | Correct source or graph `spec` fields | Source fields |
| Nested `spec` | Expected kind fields included and other tested kind's fields excluded | Source completion and hover |
| Missing `kind` | No kind-specific fields; missing-kind diagnostic | Not exercised |
| Kind changed while source fields remain | Wrong-kind fields rejected; required graph fields reported | Not exercised |

The checked-in matrix does not exercise the data-graph per-kind endpoint or a
live edit sequence in an IDE. The root kind-change case is evaluated as the
resulting changed document, which proves branch-sensitive completion and
validation but not editor UI timing.

This confirms completion quality is acceptable after `kind` is present. The
trade-off is weaker kind-specific completion before that discriminator is typed;
this is preferable to merging fields from every kind. Per-kind files remain a
useful direct endpoint and can provide earlier suggestions when a producer knows
the kind.

The committed `verify.mjs` asserts top-level/nested completion, hover, wrong-kind
validation, missing-kind behavior, and changing-kind behavior. Its output is
checked in as
[`yaml-language-server-matrix.json`](dex-1012-demo/evidence/yaml-language-server-matrix.json).
The visual artifact is
[`completion-hover.svg`](dex-1012-demo/evidence/completion-hover.svg), a
reconstructed UI illustration rather than completed VS Code acceptance proof.
The JSON schemas are explicitly illustrative fixtures, not the DEX-1000 schema
output.

## 4. JetBrains

IntelliJ Platform 2025.2 source and documentation indicate recognition of both
YAML comments:

```yaml
# yaml-language-server: $schema=https://example/schema.json
# $schema: https://example/schema.json
```

Therefore the chosen modeline is portable to current JetBrains IDEs with YAML
and JSON Schema support. JetBrains also supports Draft 7 conditional completion,
so the root-schema design applies. One precedence difference matters: a
JetBrains user mapping can override a schema comment, while yaml-language-server
gives the modeline higher priority.

Fallback for older/product-specific installations:

1. Open **Settings/Preferences | Languages & Frameworks | Schemas and DTDs |
   JSON Schema Mappings**.
2. Add the Rudder root URL as a user schema.
3. Associate it by **file path pattern** or **directory**. The editor status
   widget also exposes **New Schema Mapping…** / **Edit Schema Mappings…**.
4. Ensure downloading remote schemas is allowed, or select a downloaded release
   bundle for offline work.

Sources: [Red Hat YAML association docs](https://github.com/redhat-developer/vscode-yaml#associating-schemas),
[yaml-language-server changelog](https://github.com/redhat-developer/yaml-language-server/blob/main/CHANGELOG.md),
[JetBrains JSON Schema setup](https://www.jetbrains.com/help/idea/json.html#ws_json_schema_add_custom),
and IntelliJ Platform 2025.2
[`JsonSchemaByCommentProvider`](https://github.com/JetBrains/intellij-community/blob/idea/252.23892.409/json/backend/src/com/jetbrains/jsonSchema/impl/JsonSchemaByCommentProvider.kt)
(accessed 2026-09-25).

## Deferred implementation checklist

After DEX-1000 and DEX-1004 land:

- publish generated `root.json`, all per-kind schemas, and their references at
  the docs-hosted `/v1/` namespace; attach the bundle to releases;
- update importer and future scaffolding modelines to the stable root URL;
- add Hugo editor-setup documentation for modeline, VS Code `yaml.schemas`,
  JetBrains mapping, and offline bundles;
- propose the narrow SchemaStore entry only after URLs are live;
- coordinate RUD-2357 content detection with the same root/per-kind registry;
- rerun the committed matrix with generated schemas and verify production URL
  status, content type, CORS, caching, completion, hover, and `$ref` loading in
  VS Code and JetBrains.
