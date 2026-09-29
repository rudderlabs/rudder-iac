# Machine-readable validation (`rudder-cli validate --format json`)

`rudder-cli validate` checks your specs against the CLI's syntactic and semantic
rules. By default it prints a human-readable report. With `--format json` it emits
**structured diagnostics** — stable error codes and exact source positions —
suitable for editors, CI, and coding agents.

The human output is unchanged; JSON is opt-in.

## Usage

```bash
# Human-readable (default)
rudder-cli validate

# Machine-readable diagnostics
rudder-cli validate --format json
```

In JSON mode, stdout is a single parseable document and the process exits non-zero
when validation fails (without extra stderr noise), so it composes cleanly in
pipelines:

```bash
rudder-cli validate --format json | jq '.diagnostics[] | {code, file, line, message}'
```

## Diagnostic shape

```json
{
  "diagnostics": [
    {
      "code": "datacatalog/properties/spec-syntax-valid",
      "severity": "error",
      "message": "…",
      "kind": "properties",
      "file": "specs/properties.yaml",
      "line": 12,
      "col": 7,
      "ruleDoc": "docs/generated/rules.yaml#datacatalog/properties/spec-syntax-valid"
    }
  ]
}
```

- **`code`** is the rule's stable ID (`provider/kind/rule-name`) — the same key
  used in the generated rule catalog, so codes don't drift.
- **`line`/`col`** point at the offending node (resolved from the parsed spec).
- **`ruleDoc`** links to the rule's entry in `docs/generated/rules.yaml`
  (regenerate with `make gen-rule-docs`).

## Example use cases

- **Editor diagnostics.** An editor extension can run `validate --format json` on
  save and turn each diagnostic into an inline squiggle at `line`/`col`.
- **CI gates by code.** Fail or annotate a build on specific rule codes; parse the
  JSON instead of scraping text.
- **Agent self-correction.** An LLM/coding agent gets deterministic,
  position-anchored feedback (code + file + line) to fix a spec without guessing.

## How it works

- Validation semantics are unchanged — this adds a JSON renderer over the existing
  engine, plus a `kind` field on each diagnostic.
- Source positions come from the loader's existing path index (each spec is parsed
  via `yaml.Node`, and a diagnostic's JSON-Pointer reference resolves to a
  line/column).
- Error codes reuse the rule IDs already published in the rule catalog — one
  source of truth for both the docs and the JSON output.
