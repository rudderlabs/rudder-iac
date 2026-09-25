# JSON Schema for CLI specs

`rudder-cli` generates [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12) documents for every `rudder/v1` kind supported by the active provider configuration. The schemas are derived from the same typed provider models used to load specs, including destination-definition variants and enabled experimental RETL kinds.

## Generate schemas

```bash
# List available kinds
rudder-cli schema

# Print one kind schema
rudder-cli schema tracking-plan

# Write one schema per kind and the combined root schema
rudder-cli schema --out .rudder/schemas
```

The output directory contains `<kind>.schema.json` files and `rudder-spec.schema.json`. The root schema is kind-discriminated and accepts any generated kind. Output is deterministic and generation requires no RudderStack credentials or network access.

## Editor setup

### VS Code

Install Red Hat's YAML extension and associate the generated root schema in `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "./.rudder/schemas/rudder-spec.schema.json": ["**/*.yaml", "**/*.yml"]
  }
}
```

You can instead add a per-file modeline:

```yaml
# yaml-language-server: $schema=./.rudder/schemas/source.schema.json
version: rudder/v1
kind: source
```

### JetBrains IDEs

Open **Settings | Languages & Frameworks | Schemas and DTDs | JSON Schema Mappings**, add `.rudder/schemas/rudder-spec.schema.json`, and map it to the directories or YAML file patterns containing RudderStack specs.

## Opt-in modelines for generated files

Importer and migration writers can add a modeline to full YAML specs. This is disabled by default until a stable hosted schema service is published. Enable it by setting a base URL:

```bash
export RUDDERSTACK_CLI_SCHEMA_BASE_URL=https://example.com/rudder-cli/v1/schemas
rudder-cli import workspace
```

A generated `source` spec then references `${RUDDERSTACK_CLI_SCHEMA_BASE_URL}/source.schema.json`. Use a versioned, immutable URL whose files were produced by the matching CLI release. Modelines are not added to JavaScript, SQL, variable files, import manifests, or other non-spec content.

## Current limitations

JSON Schema includes constraints that map directly from Go models and validation tags, such as required fields, enums, constants, lengths, item counts, and numeric bounds. Custom validation rules, cross-resource references, backend capabilities, and conditional validators that cannot be represented safely remain enforced by `rudder-cli validate` rather than the editor schema.
