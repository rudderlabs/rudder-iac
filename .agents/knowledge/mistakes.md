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

## ACT2-463 — Destination Support CI Env Coupling
<!-- ticket:ACT2-463 -->
- The `test with code coverage` workflow can fail `cli/tests` destroy/dry-run when `RUDDERSTACK_X_DESTINATION_SUPPORT=true` but `RUDDERSTACK_X_UNVERIFIED_DESTINATIONS` is empty, because managed S3 destinations in the shared e2e account then load as unregistered type `S3` version 1.
- Keep CI's unverified-destinations default aligned with destination support so the S3 destination definition is registered whenever destination remote loading is active.
