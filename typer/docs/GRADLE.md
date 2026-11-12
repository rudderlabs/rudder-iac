# Using rudder-cli typer from Gradle

Generate the Kotlin bindings during the build, without checking them in and without credentials on the build machine.

The flow has two steps in two places:

1. A CI job that has credentials runs `typer snapshot` and publishes the snapshot file, for example to an artifact store.
2. The Gradle build reads that file from the project directory and runs `typer generate --snapshot`. Getting the file there is up to you. You can commit it, or add a step that downloads it before the build. This step needs no access token and makes no API calls, so it finishes in seconds.

## Step 1: save a snapshot (CI)

```bash
rudder-cli typer snapshot --tracking-plan-id <tracking-plan-id> -o tracking-plan.snapshot.json
```

To pin a specific tracking plan version instead of the latest one, add `--tracking-plan-version`:

```bash
rudder-cli typer snapshot --tracking-plan-id <tracking-plan-id> --tracking-plan-version 3 -o tracking-plan.snapshot.json
```

The same flag works on `typer generate` when it reads from the workspace. It needs a server release that returns events for a pinned version, and the CLI says so when the server does not.

The file holds the tracking plan with its event schemas, plus a `formatVersion`. It contains the workspace and user IDs the API returns, and no credentials, so you can decide where to store it. A CLI that finds a newer `formatVersion` asks you to upgrade `rudder-cli`. A file it cannot read as a snapshot, such as a raw API response, needs to be regenerated with `typer snapshot`. Snapshots do not expire, and they only change when you run `typer snapshot` again.

## Step 2: generate during the build (Gradle)

This snippet is for a Kotlin JVM module. Android modules are not covered, see below.

`build.gradle.kts`:

```kotlin
val rudderCli = providers.environmentVariable("RUDDER_CLI").orElse("rudder-cli")
val rudderCliVersion = providers.environmentVariable("RUDDER_CLI_VERSION").orElse("unpinned")
val snapshot = layout.projectDirectory.file("tracking-plan.snapshot.json")
val generatedDir = layout.buildDirectory.dir("generated/rudder/kotlin")

val generateRudderTyper by tasks.registering(Exec::class) {
    inputs.file(snapshot)
    inputs.property("rudderCli", rudderCli)
    inputs.property("rudderCliVersion", rudderCliVersion)
    outputs.dir(generatedDir)

    environment("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")

    commandLine(
        rudderCli.get(), "typer", "generate",
        "--platform", "kotlin",
        "--snapshot", snapshot.asFile.absolutePath,
        "--output", generatedDir.get().asFile.absolutePath,
        "--option", "packageName=com.example.analytics",
    )
}

// A task provider as a source directory also makes compileKotlin depend on the task
kotlin.sourceSets.getByName("main").kotlin.srcDir(generateRudderTyper)
```

Gradle skips the task when the snapshot file, the CLI version and the output directory are unchanged. The generated header contains the CLI version and no timestamp, so repeated runs write identical files.

Gradle does not look inside the executable that an `Exec` task runs, so the CLI version has to be a declared input. Set `RUDDER_CLI_VERSION` to the same value when you install the CLI, and an upgrade reruns the task. The install script accepts that variable:

```bash
export RUDDER_CLI_VERSION=<version>
curl -fsSL https://github.com/rudderlabs/rudder-iac/releases/latest/download/install.sh | sh
```

### Telemetry

The CLI sends one telemetry request per run unless `RUDDERSTACK_CLI_TELEMETRY_DISABLED=true` is set. The snippet sets it, so a build machine without outbound access does not wait on that request.

### Read-only home directory

The CLI writes its log and config under `$HOME/.rudder`, even when you pass `--config`, and exits with an error when it cannot. In a sandbox with a read-only home, give the task a writable `HOME`:

```kotlin
val rudderHome = layout.buildDirectory.dir("rudder-home")

// inside generateRudderTyper
doFirst { rudderHome.get().asFile.mkdirs() }
environment("HOME", rudderHome.get().asFile.absolutePath)
```

### Android modules

An Android module has one Kotlin compile task per variant, such as `compileDebugKotlin`, and no task named `compileKotlin`. The generated directory has to be added through the Android Gradle Plugin's variant API instead of the `srcDir` line above. That wiring is not documented or tested here yet.

## Other options

- `typer generate --local --location <dir>` reads YAML specs instead of a snapshot.
- `rudder-cli typer options kotlin` lists the Kotlin options for `--option key=value`.
