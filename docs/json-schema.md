# JSON Schema for spec kinds

rudder-cli generates a [JSON Schema](https://json-schema.org/) (Draft 2020-12)
for every supported spec `kind`. Pointing your editor at a schema gives you
inline completion, hover documentation, and per-field validation as you write
YAML specs — most of an "LSP" experience for free via
[yaml-language-server](https://github.com/redhat-developer/yaml-language-server).

The schema is **generated from the typed Go structs the CLI actually parses**,
so it can never drift from the real spec format. Adding or changing a field on a
spec struct flows into the schema automatically; there is no hand-authored
schema to keep in sync.

## The `schema` command

```bash
# List the kinds that have a schema
rudder-cli schema

# Print one kind's schema to stdout
rudder-cli schema tracking-plan

# Write every kind (plus a combined root schema) to a directory
rudder-cli schema --out ./.rudder/schemas
```

`--out` writes one file per kind (`<kind>.schema.json`) and a combined,
kind-discriminated `rudder-spec.schema.json` that validates any supported spec
file by branching on its `kind`.

## Editor setup

### Automatic (imported / scaffolded specs)

When rudder-cli writes spec files (for example during `rudder-cli import`), it
prepends a modeline pointing at the kind's schema:

```yaml
# yaml-language-server: $schema=.rudder/schemas/transformation.schema.json
version: rudder/v1
kind: transformation
metadata:
  name: my-transformations
spec:
  id: enrich_user
  # ...
```

Generate the referenced schema files once so editors can resolve them:

```bash
rudder-cli schema --out ./.rudder/schemas
```

### Manual

Add the modeline yourself as the first line of any spec file, pointing at a
schema you generated with `--out`:

```yaml
# yaml-language-server: $schema=./.rudder/schemas/events.schema.json
```

Or associate schemas by glob in your editor settings. For VS Code with the
[YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml):

```jsonc
// .vscode/settings.json
{
  "yaml.schemas": {
    "./.rudder/schemas/tracking-plan.schema.json": "**/tracking-plans/*.yaml",
    "./.rudder/schemas/events.schema.json": "**/events/*.yaml"
  }
}
```

## Verifying it works in an editor

1. Run `rudder-cli schema --out ./.rudder/schemas`.
2. Open a spec file with the `# yaml-language-server: $schema=...` header (or a
   `yaml.schemas` association) in an editor with the YAML language server.
3. Change a field to the wrong type — e.g. set a string field like `name:` to a
   number, or use an invalid `language:` value on a `transformation` — and the
   editor flags it inline. Removing a required field is likewise flagged.

## How it works

- `cli/internal/schema` reflects each kind's `spec:` struct into a Draft 2020-12
  schema and wraps it in the full spec envelope (`version` / `kind` /
  `metadata` / `spec`), pinning `kind` to a constant.
- Field constraints already declared with go-playground `validate` struct tags
  (`required`, `oneof`, `gte`/`lte` length bounds) are folded into the schema so
  it mirrors what the loader enforces. Only rules that map cleanly to JSON
  Schema are translated, so the schema is a sound over-approximation: it never
  rejects a spec the CLI would accept.
- The kind → struct mapping lives in one registry
  (`cli/internal/schema/registry.go`). Adding a new kind is a single entry
  pointing at its Go struct — never a hand-written schema file.
