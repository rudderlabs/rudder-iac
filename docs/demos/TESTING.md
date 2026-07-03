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

### Option A — download a prebuilt binary (no Docker, no Go toolchain)

The [`build test binaries`](../../.github/workflows/build-test-binaries.yml)
workflow rebuilds macOS/Linux binaries on every push to the branch and uploads
them as **workflow artifacts** — gated behind GitHub sign-in (not on a public
Releases page). Grab them with the [`gh` CLI](https://cli.github.com/) after
`gh auth login`.

This one-liner finds the latest successful build and pulls the artifact for your
OS/arch:

```bash
REPO=rudderlabs/rudder-iac
OS=$(uname -s | tr '[:upper:]' '[:lower:]')                              # darwin | linux
ARCH=$(uname -m); [ "$ARCH" = x86_64 ] && ARCH=amd64; [ "$ARCH" = aarch64 ] && ARCH=arm64
RID=$(gh run list -R "$REPO" -w "build test binaries" -s success -L1 \
        --json databaseId -q '.[0].databaseId')
gh run download -R "$REPO" "$RID" -n "rudder-cli-$OS-$ARCH"
mv "rudder-cli-$OS-$ARCH" rudder-cli && chmod +x rudder-cli
xattr -d com.apple.quarantine rudder-cli 2>/dev/null || true            # macOS: clear Gatekeeper
./rudder-cli --version
```

Artifact names, one per target:

| Platform | Artifact |
|----------|----------|
| macOS Apple Silicon | `rudder-cli-darwin-arm64` |
| macOS Intel | `rudder-cli-darwin-amd64` |
| Linux x86_64 | `rudder-cli-linux-amd64` |
| Linux ARM64 | `rudder-cli-linux-arm64` |

Put it on your `PATH` (`sudo mv rudder-cli /usr/local/bin/`) to run `rudder-cli`
instead of `./rudder-cli` below. The binaries are unsigned — on macOS the
`xattr` line above is what lets Gatekeeper run them.

### Option B — Docker image from the PR

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

### Option C — build from source

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
rough edges, missing types, naming, ergonomics. Both the `pr-633` image and the
workflow-artifact binaries are rebuilt on every push to the branch — `docker
pull` or re-run the download one-liner to get the latest.
