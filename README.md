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

To run commands with local catalog files, mount your local catalog directory and use the `-l` flag. For example:

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
