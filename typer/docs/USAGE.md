# Generating typed tracking code with typer

`rudder-cli typer` turns a tracking plan into typed code for Kotlin, Swift or TypeScript. A misspelled event name or a wrong property type then fails when you compile, not when the event reaches RudderStack.

## Commands

| Command | What it does |
| --- | --- |
| `rudder-cli typer generate` | Writes typed code for one tracking plan. |
| `rudder-cli typer snapshot` | Saves a tracking plan from your workspace to a file, so `generate` can run later without credentials. |
| `rudder-cli typer options --platform <platform>` | Lists the options that `--option key=value` accepts for a platform. |

## Where the tracking plan comes from

`typer generate` reads the plan from one of three places.

| Source | Flags | Needs an access token and network |
| --- | --- | --- |
| Your workspace | `--tracking-plan-id <id>`, optionally `--tracking-plan-version <n>` | Yes |
| A snapshot file | `--snapshot <file>` | No |
| Local YAML specs | `--local --location <dir> --tracking-plan-id <id>` | No |

```bash
rudder-cli typer generate --platform kotlin --tracking-plan-id <id> -o ./generated
rudder-cli typer generate --platform kotlin --snapshot ./tracking-plan.snapshot.json -o ./generated
rudder-cli typer generate --platform kotlin --local --location ./project --tracking-plan-id <id> -o ./generated
```

A few flag combinations are errors, so the CLI never generates from a different plan than the one you named:

- `--local` and `--snapshot` cannot be used together.
- `--tracking-plan-id` does not apply with `--snapshot`, because a snapshot holds exactly one plan.
- `--tracking-plan-version` applies only when reading from the workspace. A snapshot or local specs are already a single fixed version.

## Pin a tracking plan version

Without `--tracking-plan-version`, the CLI uses the latest version of the plan. To generate from an earlier version, pass its number:

```bash
rudder-cli typer generate --platform kotlin --tracking-plan-id <id> --tracking-plan-version 3 -o ./generated
```

The flag also works on `typer snapshot`. A pinned version needs a server release that returns events for a specific version. When the server does not, the CLI says so. A negative version is an error.

## Save a snapshot

```bash
rudder-cli typer snapshot --tracking-plan-id <id> -o tracking-plan.snapshot.json
rudder-cli typer snapshot --tracking-plan-id <id> --tracking-plan-version 3 -o tracking-plan.snapshot.json
```

The command prints the plan name, version and event count it saved. The file contains the plan, its event schemas and a `formatVersion`. It holds no credentials. A snapshot changes only when you run `typer snapshot` again, so builds that read it are repeatable. The common pattern runs `snapshot` in CI, where the token lives, and `generate --snapshot` on every build machine. [GRADLE.md](GRADLE.md) shows that flow in a Gradle build.

## Platforms and options

`--platform` takes `kotlin`, `swift` or `typescript`. `--option key=value` sets a platform option and can repeat. List the options for a platform with `typer options`:

```bash
rudder-cli typer options --platform kotlin
rudder-cli typer generate --platform kotlin --tracking-plan-id <id> \
  --option packageName=com.example.analytics --option outputFileName=Analytics.kt -o ./generated
```

An option the platform does not know is an error.

## Output

`--output` (`-o`) sets the directory, and the default is the current directory. The generated file starts with a header that names the CLI version and carries no timestamp, so generating twice from the same input writes identical files.

## Credentials

Generating from your workspace, and `typer snapshot`, use the access token from your CLI configuration, or the `RUDDERSTACK_ACCESS_TOKEN` environment variable. Generating from a snapshot or local specs needs no token. The CLI sends one telemetry request per run unless `RUDDERSTACK_CLI_TELEMETRY_DISABLED=true` is set.
