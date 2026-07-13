# Feedback

> Durable human direction, corrections, and review guidance.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> identifying the writer/PR/run; human-authored sections are conventionally left
> untouched by automated runs.

## RUD-2963 — Optional Feature Predicate Naming
<!-- ticket:RUD-2963 -->
- Use `APIError.FeatureFlagNotEnabled()` as the public predicate name when detecting either recognized HTTP 403 feature/flag-disabled response; keep client callers and tests consistent with that name.
- Describe the two recognized API message prefixes neutrally, without assigning lifecycle labels such as legacy or GA to either one.
