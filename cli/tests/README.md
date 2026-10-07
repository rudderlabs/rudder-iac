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

## Demos

Any test in this suite can, in principle, be turned into a narrated,
reproducible terminal demo from a real run — nothing is hand-written and
nothing is simulated, the demo replays the commands the test actually
executed. In practice, day one shipped only 2 of the 6; see below for why.

```bash
make demo DEMO_TEST=TestAccountsApply DEMO_PROFILE=mini   # record + generate
make demo-cast DEMO_TEST=TestAccountsApply                # record a .cast
make demo-check DEMO_TEST=TestAccountsApply               # fail if it drifted
make demo-test                                            # run every demo's checks (CI-safe, no backend)
```

The result is `demos/<Test>/` — a `demo.sh` you can run by hand, the spec
fixtures it needs, and a `manifest.json` naming the branch, commit and backend
it was recorded against.

**Prerequisites for the tooling itself.**

- [demo-magic](https://github.com/paxtonhare/demo-magic) — `demo.sh` sources
  it to get `p`/`pe` (type-and-run narration). It defaults to
  `$HOME/workspace/demo-magic/demo-magic.sh`, which only exists on the
  machine that wrote that default; clone demo-magic and set `DEMO_MAGIC` to
  point at your own copy of `demo-magic.sh` if that path doesn't exist for
  you.
- `python3` — `demo-cast` and `demo-check` both shell out to it (recording
  goes through `scripts/demo_cast.py`'s stdlib PTY recorder, not
  `asciinema rec`; drift-checking compares manifests with stdlib `json`).
  Nothing beyond the standard library is required.

**Env vars that steer a run**, named here because the prose below only
describes them: `RUDDER_DEMO_AGENT=1` makes a demo write `ANNOTATE.md` and
exit 3 for missing narration instead of printing an interactive prompt (see
below); `RUDDER_DEMO_RECORDING=1` (set for you by `demo-cast`) suppresses
that footer entirely so a prompt never lands mid-cast.

**Day one, only 2 of the 6 top-level e2e tests have a demo committed:**
`TestAccountsApply` and `TestAccountsImportWorkspace` (the latter needed
`RUN_ACCOUNT_E2E=1` exported at recording time — the test itself is skipped
without it). The other four are not committed here:

- `TestConnectionsApply` needs `RUN_CONNECTION_E2E=1`; CI already passes this
  variable to the e2e job but does not yet define it (DEX-901), so today it
  is never actually on.
- `TestDestinationsApply` makes ~225 destination writes and runs nightly
  against its own workspace instead of on every PR — not something to record
  a demo from casually.
- `TestProjectApply` and `TestTransformationsTest` carry no gate; they simply
  were not attempted against this mini instance in this pass.

Recording one of these is the same `make demo DEMO_TEST=<Test>
DEMO_PROFILE=mini`, with the relevant gate exported first if the test needs
one.

**Narration.** Step prose is derived from subtest names, so every test is
demoable with no work. Where the *reason* matters and a subtest name cannot
carry it, add one line in the test:

```go
demo.Say(t, "The API never returns write-only secrets, so the CLI re-sends them every time.")
```

It writes into the demo journal and is inert when no demo is being recorded. A
demo with no narration writes an `ANNOTATE.md` naming the steps that want it,
and exits 3 when run non-interactively, so an agent is told to add the lines
and open a PR rather than leaving the gap unnoticed.

**Verification.** A step whose last command writes rather than reads proves
itself in Go, invisibly. Those steps are listed in `demos/GAPS.md`; each one
wants a read-only CLI command that shows the result on screen. Where no such
command exists, file it — a demo that has to leave the tool it demonstrates is
a gap in the tool, not in the demo.

**Backends.** A profile is an env file under `demos/profiles/`. The CLI already
selects its backend from `RUDDERSTACK_API_URL` and `RUDDERSTACK_ACCESS_TOKEN`,
so a demo runs against rudder-mini locally and a cloud backend in CI with no
code change. The manifest records which.

---

Happy testing! 🚀
