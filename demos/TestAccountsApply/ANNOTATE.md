# Narration wanted: TestAccountsApply

These steps show what ran but not why it matters. Add a `demo.Say` call
immediately before the command in each subtest below, then re-record.

```go
	demo.Say(t, "Why this step matters.")
```

- `TestAccountsApply`
  - derived prose: "Accounts apply"
  - verification is invisible here; if a read-only CLI command can show it, run one
- `TestAccountsApply/apply_create`
  - derived prose: "Apply create"
  - verification is invisible here; if a read-only CLI command can show it, run one
- `TestAccountsApply/apply_update`
  - derived prose: "Apply update"
  - verification is invisible here; if a read-only CLI command can show it, run one
- `TestAccountsApply/re-apply_leaves_non-secret_upstream_state_unchanged`
  - derived prose: "Re-apply leaves non-secret upstream state unchanged"
  - verification is invisible here; if a read-only CLI command can show it, run one

When this is done, open a PR. `demos/GAPS.md` lists anything that needed a
command the CLI does not have yet — those are tickets, not narration.
