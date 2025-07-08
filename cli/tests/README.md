# Rudder CLI End-to-End (E2E) Tests

The E2E suite provides high confidence that changes to `rudder-iac` do not introduce regressions. It reproduces real-world CLI interactions against the data catalog and verifies that both the upstream resources and the CLI-managed state match committed snapshots.

## Overview

1. **Binary build (TestMain)** – Before the tests run, `TestMain` compiles the current source and places the resulting binary in a temporary directory. All E2E tests invoke this binary.
2. **Scenario data** – Test inputs live under `cli/tests/testdata/`:
   • `create/` contains the initial YAML definitions to apply.
   • `update/` contains subsequent modifications for the same resources.
3. **Apply flow** – For each scenario the tests run `rudder-cli tp apply` twice—once with `create/`, once with `update/`—and capture:
   • The **state file** returned by the CLI.
   • The **upstream resource payloads** fetched via the public API.
4. **Snapshot assertion** – Captured state and payloads are compared to the JSON snapshots in `cli/tests/testdata/expected/`. If they diverge, a unified diff is printed.

## Running the Suite

Make targets:

| Target          | Description             |
|-----------------|-------------------------|
| `make test`     | Unit tests only         |
| `make test-e2e` | E2E tests only          |
| `make test-all` | Unit + E2E tests        |

Prerequisites:

1. Export `RUDDERSTACK_ACCESS_TOKEN`.  
   When unset, the tests fall back to the token in `~/.rudder/config.json`.
2. Ensure your machine can reach the Control Plane.

During execution a detailed log is written to `$HOME/.rudder/cli.log` (log level **DEBUG**).

## Snapshot Verification

Two snapshot levels are asserted:

1. **CLI state** – `expected/state/`
2. **Upstream resources** – `expected/upstream/`

Each managed resource is stored as a separate JSON file named after its URN, for example `event_product_viewed_1`.

### Dynamic fields

Fields such as `id`, `createdAt`, and `updatedAt` change on every run. The comparator maintains an ignore list of JSON paths so these values do not cause false positives.

## Debugging Failures

1. Inspect the test failure message—it identifies the mismatching resource and shows a diff.
2. Review `cli.log` to trace the executed CLI commands and API calls.
3. Resources are left upstream after the run, allowing manual inspection in the test account.

## Adding or Updating Scenarios

1. Place new or updated YAML files in `cli/tests/testdata/{create|update}/`.
2. Run `rudder-cli tp apply` manually to generate the state.
3. Fetch the upstream JSON using `https://<CONTROL_PLANE>/v2/cli/catalog/state` manually (or the helpers in `cli/tests/helpers`).
4. Fetch the upstream resource JSON based on the `id` of the resource and public endpoint to view the entity.
4. Copy the JSON into `expected/state/` and `expected/upstream/`, following the URN-based filename convention.
5. Commit the snapshots and run `make test-e2e`.

---

Happy testing! 🚀
