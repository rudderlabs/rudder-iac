# Narration wanted: TestAccountsImportWorkspace

These steps show what ran but not why it matters. Add a `demo.Say` call
immediately before the command in each subtest below, then re-record.

```go
	demo.Say(t, "Why this step matters.")
```

- `TestAccountsImportWorkspace`
  - derived prose: "Accounts import workspace"
  - verification is invisible here; if a read-only CLI command can show it, run one

When this is done, open a PR. `demos/GAPS.md` lists anything that needed a
command the CLI does not have yet — those are tickets, not narration.
