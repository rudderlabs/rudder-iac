# Formatting spec files (`rudder-cli fmt`)

`rudder-cli fmt` rewrites spec YAML into a canonical form — like `gofmt` or
`terraform fmt`. It normalizes **layout only** (indentation, whitespace, quoting)
and **preserves your comments and key order**. Formatting is idempotent and
semantics-preserving: a formatted spec parses to exactly the same thing it did
before.

The point is consistency: identical specs look identical in git, so review diffs
show real changes instead of whitespace noise — and tools or LLM agents that emit
specs have a single canonical target to match.

## The `fmt` command

```bash
# Format the current directory (recursively), rewriting files in place
rudder-cli fmt

# Format specific files or directories
rudder-cli fmt ./specs spec.yaml

# CI mode: don't write; exit non-zero if any file isn't formatted
rudder-cli fmt --check ./specs

# Preview: print a unified diff of what would change, without writing
rudder-cli fmt --diff spec.yaml
```

With no path, the current directory is formatted recursively. `--check` and
`--diff` are mutually exclusive.

## Example use cases

- **Clean PR diffs.** Run `rudder-cli fmt` before committing so reviews surface
  intent, not reformatting churn.
- **CI hygiene gate.** Add `rudder-cli fmt --check .` to CI; the non-zero exit
  fails the build when a spec was committed unformatted.
- **Pre-commit hook.** Wire `rudder-cli fmt` (or `--check`) into a pre-commit hook
  to keep the repo canonical automatically.
- **Canonical output for agents.** When an LLM/coding agent generates or edits
  specs, run `fmt` afterward so its output is deterministic and diff-stable.

## How it works

- `fmt` operates on the YAML text directly (before any provider parsing) — it is a
  layout transform, not a semantic one.
- Files are parsed into a comment-preserving node tree (`yaml.Node`) and
  re-encoded with canonical indentation and spacing; comments and key order are
  kept as written.
- The transform is idempotent: running `fmt` on already-formatted output produces
  no change (covered by a property test).
