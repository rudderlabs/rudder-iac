# Using rudder-cli typer from Gradle

Generate the Kotlin bindings during the build, without checking them in and without credentials on the build machine.

The flow has two steps in two places:

1. A CI job that has credentials runs `typer snapshot` and publishes the snapshot file (for example to an artifact store or a repository).
2. The Gradle build fetches that file and runs `typer generate --snapshot`. This step needs no token and no network access to RudderStack, and it finishes in seconds because it makes no API calls.

## Step 1: save a snapshot (CI)

```bash
rudder-cli typer snapshot --tracking-plan-id <tracking-plan-id> -o tracking-plan.snapshot.json
```

To pin a specific tracking plan version instead of the latest one, add `--tracking-plan-version`:

```bash
rudder-cli typer snapshot --tracking-plan-id <tracking-plan-id> --tracking-plan-version 3 -o tracking-plan.snapshot.json
```

The same flag works on `typer generate` when it reads from the workspace. It needs a server release that returns events for a pinned version, and the CLI says so when the server does not.

The file holds the tracking plan as the API returned it, plus a `formatVersion`. A CLI refuses a snapshot with a format version it does not know and tells you to regenerate it. Snapshots do not expire, but they only change when you run `typer snapshot` again.

## Step 2: generate during the build (Gradle)

`build.gradle.kts`:

```kotlin
val rudderCli = providers.environmentVariable("RUDDER_CLI").orElse("rudder-cli")
val snapshot = layout.projectDirectory.file("tracking-plan.snapshot.json")
val generatedDir = layout.buildDirectory.dir("generated/rudder/kotlin")

val generateRudderTyper by tasks.registering(Exec::class) {
    inputs.file(snapshot)
    inputs.property("rudderCli", rudderCli)
    outputs.dir(generatedDir)

    commandLine(
        rudderCli.get(), "typer", "generate",
        "--platform", "kotlin",
        "--snapshot", snapshot.asFile.absolutePath,
        "--output", generatedDir.get().asFile.absolutePath,
        "--option", "packageName=com.example.analytics",
    )
}

kotlin.sourceSets.getByName("main").kotlin.srcDir(generatedDir)
tasks.named("compileKotlin") { dependsOn(generateRudderTyper) }
```

Gradle skips the task when the snapshot file and its output directory are unchanged. The generated header contains the CLI version and no timestamp, so repeated runs write identical files.

Pin the CLI version the same way on every machine. The install script accepts a version:

```bash
curl -fsSL https://github.com/rudderlabs/rudder-iac/releases/latest/download/install.sh | RUDDER_CLI_VERSION=<version> sh
```

## Other options

- `typer generate --local --location <dir>` reads YAML specs instead of a snapshot.
- `rudder-cli typer options kotlin` lists the Kotlin options for `--option key=value`.
