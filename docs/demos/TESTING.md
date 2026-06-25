# Try the experimental resource verbs — internal testing guide

**PR:** https://github.com/rudderlabs/rudder-iac/pull/633
**What:** a kubectl-style verb layer for `rudder-cli` — `get` / `describe` /
`delete` / `set-external-id`, a scoped `apply -f`, and accounts as a read
resource. **All of it is behind the experimental `resourceCommands` flag**, so
it's inert until you turn it on.

See the annotated GIF walkthroughs in **[DEMOS.md](DEMOS.md)** (observe → build →
adopt → guardrails).

---

## 1. Get the build

### Option A — Docker image from the PR (no Go toolchain needed)

```bash
docker pull rudderlabs/rudder-cli:pr-633
```

The image's entrypoint **is** `rudder-cli`, so pass only the subcommand. It reads
auth/config from a mounted `/root/.rudder`, and the verbs need the experimental
env vars:

```bash
docker run --rm \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  -e RUDDERSTACK_X_RESOURCE_COMMANDS=true \
  -v ~/.rudder:/root/.rudder \
  rudderlabs/rudder-cli:pr-633 get event-stream-source
```

A handy shell alias for the session:

```bash
rcli() { docker run --rm \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true -e RUDDERSTACK_X_RESOURCE_COMMANDS=true \
  -v ~/.rudder:/root/.rudder -v "$PWD":/work -w /work \
  rudderlabs/rudder-cli:pr-633 "$@"; }

rcli get account
rcli apply -f /work/sources.yaml --dry-run
```

(The `-v "$PWD":/work -w /work` mount lets `apply -f`/`-l` see your local specs.)

### Option B — build from source

```bash
git fetch origin && git checkout feat/k8s-style-imperative-commands
make build            # -> ./bin/rudder-cli
```

---

## 2. Authenticate (once)

Point the CLI at the workspace you want to test against:

```bash
rudder-cli auth login          # writes ~/.rudder/config.json
```

Use a **non-critical / sandbox workspace** — `delete` and a non-dry-run
`apply` mutate real remote resources.

## 3. Enable the experimental suite

```bash
rudder-cli experimental enable resourceCommands
```

…or per-command, ephemerally:
`RUDDERSTACK_CLI_EXPERIMENTAL=true RUDDERSTACK_X_RESOURCE_COMMANDS=true rudder-cli get ...`
(The Docker alias above already sets these.) Without it the verbs are hidden and
error with the enable instructions.

---

## 4. What to try

Follow **[DEMOS.md](DEMOS.md)** for the guided flows, or quickly:

```bash
# Discover (read-only)
rudder-cli get --help
rudder-cli get event-stream-source                 # EXTERNAL-ID · REMOTE-ID · NAME · MANAGED
rudder-cli get tracking-plan -l name="My Plan"     # label selector
rudder-cli get account                             # accounts are read-only

# Inspect one
rudder-cli get event-stream-source <id> -o yaml    # re-appliable spec
rudder-cli describe event-stream-source <id>

# Scoped apply (never deletes anything outside the file)
rudder-cli apply -f my-source.yaml --dry-run       # preview
rudder-cli apply -f my-source.yaml                 # prompts unless --confirm

# Adopt an existing unmanaged source, then manage it
rudder-cli set-external-id event-stream-source <remote-id> my-source
rudder-cli delete event-stream-source my-source    # prompts unless --confirm
```

---

## 5. Safety notes

- **`apply -f` is scoped and never deletes** out-of-scope resources — contrast
  with `apply --location` (whole-project reconcile, which can prune).
- **`delete` and non-dry-run `apply` hit the real workspace.** Prefer
  `--dry-run` first; both prompt unless you pass `--confirm`.
- **Accounts are read-only** — `delete`/`set-external-id` on `account` are
  refused by design.

## 6. Feedback

Drop comments on **[PR #633](https://github.com/rudderlabs/rudder-iac/pull/633)** —
rough edges, missing types, naming, ergonomics. The image tag `pr-633` is
rebuilt on every push to the branch, so `docker pull` again to get the latest.
