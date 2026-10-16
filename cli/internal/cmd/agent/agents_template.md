# RudderStack (rudder-cli)

<!--
Written by `rudder-cli agent init` (rudder-cli {{RUDDER_CLI_VERSION}}). Edit it freely.
Rerunning the command refuses to overwrite this file unless you pass --force.

This file carries the happy path and the traps that break it. If your agent supports
Agent Skills, the full set goes deeper:
  npx skills add rudderlabs/rudder-agent-skills
-->

This project manages RudderStack resources declaratively. YAML specs describe the desired
state and `rudder-cli apply` reconciles a RudderStack workspace to them, the way Terraform does.

`rudder-cli agent init` wrote only this file. It created no spec and no resource, and it did
not apply anything.

## Credentials

The CLI reads an access token from `RUDDERSTACK_ACCESS_TOKEN` or from `~/.rudder/config.json`
(written by `rudder-cli auth login`). The token is a secret. Never read the config file, print
the token, or copy it into this project. If a command fails with "access token is required",
ask the user to run `rudder-cli auth login` themselves. Do not supply a token on their behalf.

The write key of a source is not available from the CLI. Read it from the source's page in the
RudderStack dashboard. The data plane URL is the `dataPlaneURL` field of
`rudder-cli workspace info --json`.

## Before anything else

```bash
rudder-cli workspace info --json
```

Confirm the workspace name before any `apply`. The CLI targets the one workspace that the token
belongs to, and `apply` prints the same name as a banner before it plans anything.

## The loop

Run these in order. Do not skip to `apply`.

```bash
rudder-cli validate -l <specs-dir>                  # offline, needs no token
rudder-cli apply --dry-run -l <specs-dir>           # prints the plan, changes nothing
rudder-cli apply --confirm=false -l <specs-dir>     # executes the plan
```

`--confirm=false` is required whenever stdout is not a terminal. Without it, `apply` prints the
plan, does not ask, and exits 0 without applying anything.

Show the user the `--dry-run` plan and get agreement before the real `apply`. Plans can contain
deletes, and they are real.

Commit the specs after every successful `apply`. To undo a bad apply, `git revert` the commit
and `apply` again.

## Where the specs go

Keep every spec under one `rudderstack/` directory at the project root and pass
`-l ./rudderstack` to every command:

```
rudderstack/sources/<id>.yaml          kind: event-stream-source
rudderstack/destinations/<id>.yaml     kind: destination
rudderstack/connections.yaml           kind: event-stream-connections
```

**The CLI loads every `.yaml` and `.yml` file under the `-l` directory, recursively, and
rejects any file that is not a RudderStack spec.** It does not skip `node_modules` or dot
directories. In an application repository, `-l .` therefore fails on `.github/workflows/*.yml`,
`docker-compose.yaml` and YAML inside `node_modules`, with errors named
`project/spec-syntax-parse-valid`. Files ending in `.vars.yaml` or `.vars.yml` are skipped, so
`secrets.vars.yaml` can sit at the project root and be passed with `--var-file`.
Pointing `-l` at a single file also works.

## Spec shape

Every spec file has four top-level keys.

```yaml
version: rudder/v1     # rudder/0.1 and rudder/v0.1 are legacy and print a deprecation warning
kind: event-stream-source
metadata:
  name: web-app
spec:
  id: web-app
  name: Web app
  type: javascript
  enabled: true
```

| File | `kind` | Common mistake |
|---|---|---|
| `rudderstack/sources/<id>.yaml` | `event-stream-source` | `source` |
| `rudderstack/destinations/<id>.yaml` | `destination` | |
| `rudderstack/connections.yaml` | `event-stream-connections` | `connection`, singular |

`rudderstack/connections.yaml` holds a list under `spec.connections`. A second destination is a second entry
in that list, not a second file. Each entry has `id`, `source`, `destination` and `enabled`.
There is no `config` key on a connection. Connection settings live on the destination.

A connection references its endpoints by kind and id, not by file:

```yaml
spec:
  connections:
    - id: web-app-to-webhook
      source: "#event-stream-source:web-app"
      destination: "#destination:my-webhook"
      enabled: true
```

A `destination` spec has `id`, `display_name`, `type`, `definition_version`, `enabled` and
`config`. The keys inside `config` depend on `type`. A minimal webhook destination:

```yaml
version: rudder/v1
kind: destination
metadata:
  name: events-webhook
spec:
  id: events-webhook
  display_name: Events webhook
  type: webhook
  definition_version: 1
  enabled: true
  config:
    webhook_url: https://hooks.example.com/rudderstack
    connection_mode:
      web: cloud
```

`webhook_url` must be a public http(s) URL. `localhost` and `ngrok.io` hosts fail validation.
A connection also needs a `connection_mode` entry for the source's platform in the
destination's `config`. A `javascript` source is platform `web`, and without the entry
`validate` fails with `event-stream/connection/semantic-valid`.
Each destination type has a README with its config keys in the rudder-iac repository, at
`cli/internal/providers/destination/definitions/<type>/README.md`.

Never infer a `kind` or a `type` from a filename. Copy the shape from an existing spec of the
same kind.

Validation failures name the rule that fired, for example `error[project/spec-syntax-parse-valid]`.
Fix the spec the rule points at. Run `validate` after every edit. It touches no workspace.

## Reading CLI output

`apply`, `validate` and `destroy` have no `--json` flag and exit 0 or 1. Read their text output.

These commands take `--json`, and a script or an agent should always pass it:

```bash
rudder-cli workspace info --json
rudder-cli workspace event-stream-sources list --json
rudder-cli workspace accounts list --json
rudder-cli workspace tracking-plans list --json
rudder-cli workspace retl-sources list --json
```

## Things that will bite you

- **`apply` deletes.** It removes workspace resources that the CLI manages (created by the CLI
  or imported into it) and that are absent from the local specs. There is no flag that scopes
  an apply to one resource. Read every `delete` in the `--dry-run` plan.
- **Adopt before you create.** If the workspace already has resources, run
  `rudder-cli import workspace -l <project-dir>` before writing specs. It writes the specs under
  `<project-dir>/imported/` and refuses if that directory already exists.
- **Secrets do not belong in specs.** Write `'{{ .VAR }}'` in the spec, keep the values in a
  `*.vars.yaml` file outside version control, and pass it with `--var-file`. The error
  "variable substitution failed ... passed with --var-file" means a var file is missing from
  the command.
- **Nothing in the CLI sends or reads events.** `rudder-cli` has no command that reads event
  metrics. Verify delivery at the destination's own receiving end, in the dashboard, or through
  the RudderStack MCP server if one is configured.
- **The write key is usable a short while after `apply`.** Sending an event immediately can
  return `401 invalid write key`. Wait about two minutes before retrying, and stop after a few
  attempts.

## Typed event clients

`rudder-cli typer generate --tracking-plan-id <id> --platform typescript` generates a typed
client from a tracking plan. The TypeScript client takes a function that returns the SDK
instance, not the instance itself, so each call looks up the current object. If you capture `window.rudderanalytics` at import time while the SDK is still
loading, you hold the loader stub and events are never sent, with no error.

## Going deeper

```bash
npx skills add rudderlabs/rudder-agent-skills
```

Nothing in this file depends on that.
