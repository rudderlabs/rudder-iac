# rudder-cli

## Table of Contents

- [⚠️ Work in Progress](#️-work-in-progress)
- [Installation](#installation)
  - [Install script](#install-script)
  - [Manual download](#manual-download)
    - [MacOS](#macos)
      - [Apple Silicon](#apple-silicon)
      - [Intel-based](#intel-based)
    - [Linux](#linux)
  - [Docker](#docker)
  - [Build from Source](#build-from-source)
- [Telemetry](#telemetry)

## ⚠️ Work in Progress

> **Warning**
>
> Please note that this tool is currently a work in progress. We are actively developing and improving it, and as such, there may be frequent changes and updates. We do not guarantee backwards compatibility at this stage.

## Installation

### Install script

On macOS and Linux, this command installs the latest release:

```sh
curl -fsSL https://github.com/rudderlabs/rudder-iac/releases/latest/download/install.sh | sh
```

Each release from 0.29.0 on has its own copy of the script, which installs that release:

```sh
curl -fsSL https://github.com/rudderlabs/rudder-iac/releases/download/v0.29.0/install.sh | sh
```

For about five minutes after a new release is published, its files are still uploading and the first command fails with a 404. Run it again after that.

The script picks the release archive for your OS and CPU, checks it against the release's checksums file, and installs `rudder-cli` into `~/.local/bin` without `sudo`. It runs the new binary once before it replaces anything, so a release that cannot run on your system leaves an existing `rudder-cli` in place. If that directory is not on your `PATH`, the script prints the line to add to your shell profile. If another `rudder-cli` comes earlier on your `PATH`, such as one that the manual steps below put in `/usr/local/bin`, the script names the copy that runs and prints the same line, which puts the new one first.

Set these variables on the `sh` side of the pipe to change the defaults:

- `RUDDER_CLI_VERSION` installs a specific release, such as `0.27.0` or `v0.27.0`, including releases from before 0.29.0. The default is the release the script belongs to, which for the first command above is the latest.
- `RUDDER_CLI_INSTALL_DIR` installs into a different directory.

```sh
curl -fsSL https://github.com/rudderlabs/rudder-iac/releases/latest/download/install.sh | RUDDER_CLI_VERSION=0.27.0 RUDDER_CLI_INSTALL_DIR="$HOME/bin" sh
```

The script does not support Windows. Download the `.zip` for your CPU from the [latest release](https://github.com/rudderlabs/rudder-iac/releases/latest) instead.

### Manual download

Each release also publishes the archives directly. Download the one for your platform and move the binary onto your `PATH`.

#### MacOS

##### Apple Silicon

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

##### Intel-based

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

#### Linux

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Docker

You can run the CLI directly using Docker:

```sh
docker run rudderlabs/rudder-cli
```

If you need to persist your configuration, or provide an external configuration file, you can mount your local config directory into the container. Assuming your config directory is located at `~/.rudder`, you can run the following command:

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

To run commands with local catalog files, mount your files directory and use the `-l` flag. For example:

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

This will use the access token from your local configuration file, and the catalog files from the `/catalog` directory. Alternatively, you can use the `RUDDERSTACK_ACCESS_TOKEN` environment variable to provide the access token:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

### Build from Source

To build the `rudder-cli` from source, you need to have Go installed. Then, run the following commands:

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

To build the Docker image locally:

```sh
make docker-build
```

## Telemetry

`rudder-cli` sends usage data to RudderStack, so we can see which commands are used, where they fail, and how much of that use comes from coding agents. Telemetry is on until you turn it off. The CLI prints a notice the first time it runs with telemetry on.

### What is sent

One event for each command run:

| Field | Example |
| --- | --- |
| Command | `apply` |
| Whether it failed, and at which stage | `errored: true`, `stage: pre_run` |
| The flags the command reports, with their values. A value can be a path, such as the project location, or a resource id | `location: ./rudder`, `dryRun: true` |
| Experimental features that are on | |
| CLI version | `1.4.0` |
| Telemetry id, a random UUID made on this machine and kept in `~/.rudder/config.json` | |
| CI service, when the CLI runs in one | `github-actions` |
| Coding agent running the CLI, when one is. Read from `RUDDERSTACK_CLI_AGENT`, `AI_AGENT`, or the variable the agent sets in its shells | `claude-code` |
| Skill the agent follows, when the skill sets `RUDDERSTACK_CLI_SKILL` | `rudder-setup` |

While telemetry is on, each call to the RudderStack API also carries the agent, the skill and the telemetry id in its `User-Agent` header:

```
rudder-cli/1.4.0 (agent=claude-code; skill=rudder-setup; telemetry-id=6b72f5e1-9af8-4544-92b4-67b2b6638582)
```

The API knows the workspace from your access token, so this ties the events above to your workspace.

### What is never sent

The contents of your spec files, your access token, and the values in your variable files.

### Turn it off

```sh
rudder-cli telemetry disable   # saved in ~/.rudder/config.json
rudder-cli telemetry status
```

Or set `RUDDERSTACK_CLI_TELEMETRY_DISABLED=true` or `DO_NOT_TRACK=1` in the environment. With telemetry off, the CLI sends no events and the `User-Agent` is `rudder-cli/<version>`.

A binary you build from source has no telemetry write key, so it sends no events. Its API calls still carry the `User-Agent` above unless telemetry is off.
