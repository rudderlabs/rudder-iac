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

## DEX-498 — Transformation Fixture Default-Events CI Failure
<!-- ticket:DEX-498 -->
- CI failed when `cli/tests/testdata/project/transformations-test/setup/py_transform.yaml` omitted an explicit `tests` block, causing `apply` batch publish validation to fall back to the built-in `default-events` suite.
- The failure surfaced in `TestTransformationsTest` as `Py Transform` / `default-events` returning `Internal server error` and `batch publish validation failed`.
- Durable mitigation: transformation setup fixtures should define deterministic input/output test suites so publish validation does not depend on broad default-event payloads.

## DEX-525 — Idempotent API Transport Retry Boundary
<!-- ticket:DEX-525 -->
- CI live CLI tests failed after read-only Public API requests returned transient transport errors (`read: connection reset by peer`) for catalog categories and workspace GET calls, leaving live state half-applied and causing later assertions such as missing `test-results.json` and missing `No changes to apply`.
- Durable mitigation: retry known transient transport errors only for idempotent API methods (`GET`, `HEAD`, `OPTIONS`) in `api/client.Client.Do`; do not retry mutating methods because the request may already have reached the server.

## DEX-677 — Destination Handler Test HTTP Transport Isolation
<!-- ticket:DEX-677 -->
- CI/race coverage exposed that `cli/internal/providers/destination/handler_test.go` test clients built with `client.New` but without `WithHTTPClient` can fall back to the process-global default HTTP transport, causing parallel httptest-backed destination handler tests to interfere with each other.
- The observed failure was `TestHandlerImpl_Import_TranslatesAPITypeToLocal` intermittently failing on the final `/external-id` PUT with `http: CloseIdleConnections called`.
- Durable mitigation: give each destination handler test client its own `http.Transport` through `client.WithHTTPClient(&http.Client{Transport: transport})` and close that transport from the same test cleanup.
- CI E2E exposed live catalog read-after-write lag in `TestProjectApply`: immediately after a successful migrated update apply, upstream verification saw only 25 of 40 resources and the following dry-run still reported properties/tracking plans as new.
- Durable mitigation: poll catalog-backed snapshot and no-diff dry-run assertions for a short consistency window after apply, preserving the original drift signal if eventual consistency does not settle.
- CI E2E dry-runs can fail when shared disposable workspaces contain managed unverified destinations but the global apply/destroy/dry-run path only enables destination support; the observed error was an unregistered `ATTENTIVE_TAG` destination type during destination remote loading.
- Durable mitigation: e2e tests that may run with destination support against a shared workspace should also enable `RUDDERSTACK_X_UNVERIFIED_DESTINATIONS` so remote state loading can decode unverified managed destinations left by destination e2e.
- In the test-with-coverage workflow, `RUDDERSTACK_X_DESTINATION_SUPPORT` is passed to every `cli/tests` E2E command; the unverified-destination decode risk applies to `AccountsApply`, `ProjectApply`, `TransformationsTest`, and opt-in `AccountsImportWorkspace`, not only destination-specific tests.
- Keep the production destination handler strict on unknown managed types; fix shared-workspace E2E setup so it can decode unverified destination residue instead of weakening production unknown-type errors.

## DEX-509 — Kafka Destination Fixture Snapshot Count Mismatch
<!-- ticket:DEX-509 -->
- CI failed when Kafka expected upstream destination snapshots were added under `cli/tests/testdata/expected/upstream/destinations/{create,update}/` without matching Kafka fixture YAML under `cli/tests/testdata/destinations/{create,update}/`.
- The live `TestDestinationsApply` snapshot tester reported `resource count mismatch: got 45 managed destinations, want 48 resources`, showing that fixture and expected snapshot counts must stay exactly matched.
- Durable mitigation: defer live destination snapshots by omitting both Kafka fixtures and snapshots until they can be captured together in an explicitly disposable live destination-enabled workspace.
