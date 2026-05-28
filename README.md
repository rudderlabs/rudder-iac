# rudder-iac

`rudder-iac` ships `rudder-cli`, the CLI for managing RudderStack infrastructure as code.

> The CLI is actively developed, so command behavior and release artifacts may change.

## Install

### Prebuilt binary

#### macOS, Apple Silicon

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

#### macOS, Intel

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

Run the image directly:

```sh
docker run rudderlabs/rudder-cli
```

Persist your config by mounting `~/.rudder`:

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

Run commands against local project files:

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli apply --dry-run -l /catalog
```

Use an access token instead of local config:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli apply --dry-run -l /catalog
```

### Build from source

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

## Quick start

1. Authenticate with `rudder-cli auth login`, or set `RUDDERSTACK_ACCESS_TOKEN`.
2. Validate a project with `rudder-cli validate -l ./path/to/project`.
3. Apply changes with `rudder-cli apply -l ./path/to/project`.
4. Use `rudder-cli <command> --help` for command-specific flags.

## Common commands

- `rudder-cli validate` checks project configuration.
- `rudder-cli apply` plans and applies changes.
- `rudder-cli destroy` removes managed resources.
- `rudder-cli migrate` upgrades legacy specs.
- `rudder-cli import workspace` imports an existing RudderStack workspace.
