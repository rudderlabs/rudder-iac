# Mistakes

> Post-mortem entries from observed failures: CI failures, reverts on prior PRs,
> prod incidents. Accrues over time — bootstrap leaves this empty.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.

## RUD-2752 — Memory Capture Quoting Pitfall
<!-- ticket:RUD-2752 -->
- When recording harness memory via shell heredocs/commands, unescaped backticks can trigger command substitution and silently corrupt stored notes.
- Durable mitigation: avoid backticks in shell-fed memory text or escape them explicitly so command names and literals are preserved verbatim.

## DEX-623 — Accounts Import E2E First CI Failure
<!-- ticket:DEX-623 -->
- CI proved `TestAccountsApply` can run ungated, but `TestAccountsImportWorkspace` failed on its first completion attempt at `cli/tests/command_accounts_import_test.go:230` because `/tmp/.../imported/import-manifest.yaml` was missing.
- Durable mitigation for account E2E enforcement is to keep `TestAccountsApply` in the normal CI path while leaving `TestAccountsImportWorkspace` gated by `RUN_ACCOUNT_E2E` until a follow-up fixes the missing manifest path with CI proof.

## DEX-624 — Accounts Import Manifest Failure Resolution
<!-- ticket:DEX-624 -->
- The earlier missing `imported/import-manifest.yaml` account import E2E failure was tied to import merge being disabled; after import merge became unconditional, `TestAccountsImportWorkspace` is expected to run ungated rather than remain behind `RUN_ACCOUNT_E2E`.
