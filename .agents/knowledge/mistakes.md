# Mistakes

> Post-mortem entries from observed failures: CI failures, reverts on prior PRs,
> prod incidents. Accrues over time — bootstrap leaves this empty.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.

## DEX-358 — Baseline Test Failures Outside Scope
- Repository-wide `make test` had pre-existing failures outside `varsubst` scope during DEX-358 work: `cli/internal/typer/generator/core` (`TestFileManager_AtomicOperations`) expected an error but received `nil`, and `cli/pkg/exp/project` (`TestProjectLoad`) depended on `RUDDERSTACK_ACCESS_TOKEN` being set.
- Treat these as baseline failures when evaluating DEX-358 regressions; they should not be attributed to variable-substitution package changes without independent reproduction.
