# rudder-cli

`rudder-cli` is RudderStack's infrastructure-as-code CLI for managing workspace resources declaratively with YAML. It can import existing resources, validate local specifications, preview changes, and apply them to a workspace.

## Installation

Download and install the latest release for your platform.

### macOS (Apple Silicon)

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### macOS (Intel)

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Linux (x86-64)

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Docker

```sh
docker run --rm -e RUDDERSTACK_ACCESS_TOKEN -v "$HOME/.rudder:/.rudder" -v "$PWD:/project" rudderlabs/rudder-cli validate -l /project
```

### Build from source

Install Go, then run:

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

## Quickstart

```sh
mkdir my-rudder-project && cd my-rudder-project
rudder-cli auth login
rudder-cli import workspace
rudder-cli validate
rudder-cli apply --dry-run
```

## Documentation

- [rudder-cli documentation](https://www.rudderstack.com/docs/dev-tools/rudder-cli/)
- [Validation rules](https://www.rudderstack.com/docs/dev-tools/rudder-cli/validations/)
- [Breaking changes](BREAKING_CHANGES.md)
- [Contributing](CONTRIBUTING.md)
