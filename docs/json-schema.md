# JSON Schema for CLI specs

`rudder-cli` generates [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)
documents for the YAML spec kinds exposed by the active providers. The schemas
provide editor completion and early structural validation without replacing
`rudder-cli validate`.

## Generate schemas

```sh
# List supported kinds
rudder-cli schema

# Print one kind to stdout
rudder-cli schema tracking-plan

# Write all per-kind schemas and the combined root schema
rudder-cli schema --out .rudder/schemas

# Replace existing schema artifacts explicitly
rudder-cli schema --out .rudder/schemas --overwrite
```

Per-kind files are named `<kind>.schema.json`. The combined
`rudder-spec.schema.json` uses `oneOf` branches discriminated by `kind`, so one
mapping can cover a directory containing different spec kinds. `--out` creates
the target directory, but refuses to replace any existing artifact unless
`--overwrite` is supplied. A kind argument is stdout-only and cannot be combined
with `--out`.

The generated set contains every active provider kind plus the built-in
`import-manifest` project spec. The manifest is loaded before provider dispatch,
so its dedicated project provider owns that schema outside the resource-provider
composite.

The command is local-only and does not require a RudderStack access token.
Experimental provider flags affect the generated set: for example, enabling the
RETL table or connection flags adds their corresponding kinds.

## Add modelines to generated YAML

Workspace import and project migration can opt in to a per-kind
`yaml-language-server` modeline:

```sh
rudder-cli import workspace --schema-modeline
rudder-cli migrate --schema-modeline
```

The resulting header points to the matching schema in the stable, spec-versioned
RudderStack docs namespace:

```yaml
# yaml-language-server: $schema=https://www.rudderstack.com/docs/schemas/rudder-cli/v1/events.schema.json
version: rudder/v1
kind: events
```

Use `--schema-url-base` to replace this root; this also enables the modeline.
The custom root is used exactly as supplied, without appending the CLI version:

```sh
rudder-cli import workspace \
  --schema-url-base https://schemas.example.com/rudder-cli/v1
```

The same override can be set for import and migration through
`RUDDERSTACK_CLI_SCHEMA_BASE_URL`.

Modelines are off by default. They are added only to YAML entities that are full
CLI specs whose kinds are supported by the active providers. Unknown kinds,
variable files, SQL, text artifacts, and the generated import manifest are not
annotated. YAML comments remain valid input to the project loader.

## VS Code

Install the [Red Hat YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml).
A modeline is sufficient. Alternatively, generate local schemas and map files in
`.vscode/settings.json`:

```jsonc
{
  "yaml.schemas": {
    "./.rudder/schemas/rudder-spec.schema.json": [
      "**/*.yaml",
      "**/*.yml"
    ]
  }
}
```

Use narrower globs if the repository also contains non-Rudder YAML.

## JetBrains IDEs

1. Run `rudder-cli schema --out .rudder/schemas`.
2. Open **Settings/Preferences → Languages & Frameworks → Schemas and DTDs → JSON Schema Mappings**.
3. Add a schema, select `.rudder/schemas/rudder-spec.schema.json`, and map it to
   the directory or file pattern containing RudderStack specs.

JetBrains uses its mapping UI rather than the yaml-language-server modeline, so
local generation is the most predictable setup.

## Scope and limitations

Schemas are reflected from the same provider-owned Go spec models used by the
CLI. Validator tags with direct JSON Schema equivalents are included:
`required`, `oneof`, `eq`, and type-appropriate `min`/`max` or `gte`/`lte`
bounds. Recursive `$defs` references are preserved.

Cross-field checks, named/custom validators, remote lookups, graph constraints,
and semantic relationships do not always have a sound JSON Schema equivalent.
The generated schemas intentionally omit those checks rather than rejecting
valid templated or imported specs. Run `rudder-cli validate` for authoritative
CLI validation.

Each generated schema's `$id` uses the canonical docs-hosted `/v1/` URL. CLI
releases also attach the same standalone `*.schema.json` files as an offline
fallback. Release publication enables all distributable experimental providers,
so flag-gated rETL kinds are available as assets even though a default local
generation omits them. Automatic SchemaStore publication is not part of this
change.
