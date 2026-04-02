# rudder-cli

Command-line tool for RudderStack infrastructure-as-code: declarative YAML specs for sources, destinations, connections, tracking plans, and related resources.

**Note:** The CLI is under active development; behavior and compatibility may change between releases.

## Install

**macOS (Apple Silicon)**

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

**macOS (Intel)**

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

**Linux (x86_64)**

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

## Docker

```sh
docker run rudderlabs/rudder-cli
```

Use a mounted config directory (e.g. `~/.rudder`) so the CLI can read saved credentials:

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli <command>
```

For commands that need local spec files, mount a catalog directory and pass `-l`:

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

You can set `RUDDERSTACK_ACCESS_TOKEN` instead of mounting config.

## Build from source

Requires [Go](https://go.dev/dl/). From the repository root:

```sh
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

Build the container image locally: `make docker-build`.
