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

## DEX-497 — Live E2E Command Timeout Boundary
<!-- ticket:DEX-497 -->
- CI live E2E can exceed a 2-minute per-command `cli/tests` executor timeout while apply is still making valid progress; the observed `TestProjectApply/rudder/v1_specs_after_migration/should_update_entities_in_catalog_from_project` failure was `signal: killed` after about 125s with many successful catalog/transformation updates already printed.
- Durable mitigation: keep live E2E CLI command timeouts above the normal slow-apply window and report deadline timeouts explicitly from `cli/tests/helpers.go`, so future failures distinguish real apply errors from the test helper killing a long-running command.

## DEX-702 — HubSpot Re-Onboarding CI Failures
<!-- ticket:DEX-702 -->
- CI live E2E cleanup can fail when workspace-wide destroy sees event-stream sources and destinations but event-stream connection support is disabled; the symptom is an HTTP 400 while deleting a source or destination with active connections. Durable mitigation: enable `RUDDERSTACK_X_CONNECTION_SUPPORT` alongside destination and unverified-destination support before any live `cli/tests` cleanup invokes `rudder-cli destroy`.
- CI live E2E cleanup can still fail with connection support enabled if the shared disposable workspace contains unmanaged event-stream connections with no `externalId`. Durable mitigation: live workspace cleanup should delete all event-stream connections, managed or unmanaged, before invoking `rudder-cli destroy`, while production remote loading remains limited to external-ID-managed connections.
- The lint workflow can fail before linting when `golangci/golangci-lint-action` config verification fetches the remote golangci-lint JSON schema and times out. Durable mitigation: set `verify: false` in the lint workflow while keeping the pinned golangci-lint run enabled.
- HubSpot live destination create returns backend default `config.authorizationType = "newPrivateAppApi"` for `newApi` creates even though local YAML/definition should omit `authorization_type`. Durable mitigation: keep `authorization_type` out of local config while including `authorizationType: "newPrivateAppApi"` in the HS create upstream snapshot; update snapshots may omit it when the live update response omits it.

## DEX-490 — Amplitude Event Filtering Snapshot Default
<!-- ticket:DEX-490 -->
- CI live destination E2E showed `destination:am-minimal` update responses omit default `config.eventFilteringOption` when local YAML does not set `event_filtering`; hand-written snapshots expecting `"eventFilteringOption": "disable"` failed with a missing-key mismatch.
- Durable mitigation: keep discriminator-derived default keys out of minimal Amplitude update snapshots unless the live API actually returns them, while still snapshotting create/update discriminator values when event filtering is explicitly configured.

## DEX-730 — GA Delete Account Fixture Requires Real Account
<!-- ticket:DEX-730 -->
- CI failed in `TestDestinationsApply` when the GA destination live E2E fixture set `config.rudder_delete_account_id` to placeholder `rudderCliE2eDeleteAccount`; the backend validates `rudderDeleteAccountId` against workspace accounts and returned HTTP 400 `Account not found with given id in the workspace` during `destination:ga` creation.
- Durable mitigation: keep workspace-specific GA delete-account IDs out of live destination fixtures unless the referenced account is provisioned for that CI workspace; rely on unit conversion tests for this field or inject a real workspace account ID explicitly.

## DEX-531 — Webhook Header Secret Placeholder CI Failure
<!-- ticket:DEX-531 -->
- CI failed in the `upload coverage to codecov` job when `TestWebhookHeaderSecretsAreWrappedRevealedAndMasked` expected the old shared nested-header secret placeholder (`WEBHOOK_PRODUCTION_HEADERS_TO`) while webhook export intentionally emitted indexed placeholders such as `WEBHOOK_PRODUCTION_HEADERS_0_TO` for collection elements.
- Durable mitigation: keep webhook nested-secret export assertions aligned with indexed variable naming so multi-header secrets remain distinct and `make test-all` coverage upload does not fail on stale placeholder expectations.

## DEX-735 — Google Analytics Account Fixture CI Failure
<!-- ticket:DEX-735 -->
- CI failed when legacy Google Analytics destination E2E fixtures set `rudder_delete_account_id: rudderCliE2eDeleteAccount` without seeding that account in the workspace; create/update calls returned HTTP 400 `Account not found with given id in the workspace`.
- Durable mitigation: keep `rudder_delete_account_id` out of `cli/tests/testdata/destinations/{create,update}/ga.yaml` unless E2E setup provisions a real matching account, and update `destination_ga` upstream snapshots in the same scoped change.

## DEX-731 — Live Apply/Destroy Eventual Consistency
<!-- ticket:DEX-731 -->
- CI live E2E can observe stale upstream state immediately after project apply/destroy moves through the composite provider; observed symptoms included catalog snapshot counts such as 0 resources instead of 37 and a follow-up dry-run still listing a transformation deletion like `transformation:py_transform`.
- Durable mitigation: poll the existing exact live E2E assertions for a bounded consistency window after apply/destroy operations instead of treating the first immediate remote read as definitive.
