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
