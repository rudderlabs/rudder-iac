# rudder-cli demos

Terminal recordings of the kubectl-style verb layer (`get` / `describe` /
`delete` / `set-external-id` + scoped `apply -f`, plus accounts as a read
resource). Built with [charmbracelet/vhs](https://github.com/charmbracelet/vhs)
via the `create-vhs` skill (`.claude/skills/create-vhs/`).

**▶ Watch the demos with embedded GIFs: [DEMOS.md](DEMOS.md).** This file covers
how the recordings are produced and regenerated.

> The verb suite is **experimental** — gated behind the `resourceCommands` flag.
> Enable with `rudder-cli experimental enable resourceCommands` (the demos set
> `RUDDERSTACK_CLI_EXPERIMENTAL=true` + `RUDDERSTACK_X_RESOURCE_COMMANDS=true` in
> hidden setup).

Each demo dir holds the committed **`scenes.config.ts`** (source of truth). The
`tape.tape` and `recording.{gif,mp4}` are generated and git-ignored.

| Demo | What it shows | Mutates? |
|---|---|---|
| [`20260625-observe`](20260625-observe/scenes.config.ts) | Read-only tour: `get --help`, `get <type>`, `-l` selector, `--managed`, accounts | no |
| [`20260625-build`](20260625-build/scenes.config.ts) | Full managed lifecycle of one source: dry-run → `apply -f` → `get` → `get -o yaml` → `describe` → update → `delete` | **yes** (creates + deletes a throwaway `demo-orders-source`; self-cleans) |
| [`20260625-adopt`](20260625-adopt/scenes.config.ts) | `set-external-id` adopts an **unmanaged** source: seed an id-less source via the api → `get` (MANAGED=no) → `set-external-id` → `get` (MANAGED=yes) → `describe` → `delete` | **yes** (seeds + adopts + deletes a throwaway; self-cleans) |
| [`20260625-guardrails`](20260625-guardrails/scenes.config.ts) | Capability gating, unknown-type errors, `apply -f` vs `--location`, `set-external-id` help, `workspace … list` deprecation | no (fail-fast / read-only) |

## Staging unmanaged resources

The CLI can't create an *unmanaged* (external-id-less) source — that's the whole
point of `set-external-id`. [`scripts/seed-unmanaged`](scripts/seed-unmanaged/main.go)
does it directly via the `api` client (the create endpoint rejects an empty
externalId but accepts an **omitted** one), so the adopt demo has a real orphan
to claim:

```bash
go run ./docs/demos/scripts/seed-unmanaged -name "Legacy Orders Source"  # prints remote id
go run ./docs/demos/scripts/seed-unmanaged -delete <remote-id>           # cleanup
```

## Regenerate

Both GIF and MP4, for every demo, from the repo root:

```bash
bash docs/demos/record-demos.sh
```

…or one demo:

```bash
bash docs/demos/record-demos.sh docs/demos/20260625-build
```

The script builds `./bin/rudder-cli` + `./bin/seed-unmanaged`, compiles each
`scenes.config.ts` to a `tape.tape`, records the **GIF live** with `vhs`, then
**transcodes the GIF to MP4** with `ffmpeg`. (Recording MP4 live starves ttyd and
scrambles keystrokes on longer demos, so we transcode instead — it also avoids
running a mutating demo twice.) Finally it runs `cleanup.sh` to remove any
throwaway left behind by a recording race. Override `NODE_BIN` / `VHS_BIN` if your
toolchain lives elsewhere.

Manual cleanup anytime: `bash docs/demos/cleanup.sh`.

Prereqs: `vhs`, `ttyd`, `ffmpeg` (mp4), and node ≥ 18. Check with
`bash .claude/skills/create-vhs/scripts/check-prereqs.sh`.

## Notes

- `build` records against the **live workspace** in `~/.rudder/config.json`. It
  only ever touches `demo-orders-source` and deletes it at the end.
- If `vhs` fails with `ERR_CONNECTION_REFUSED`, just re-run — it's a transient
  ttyd startup race.
