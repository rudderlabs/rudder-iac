# rudder-cli

`rudder-cli` brings Infrastructure as Code workflows to RudderStack. It lets you import workspace resources as declarative YAML, validate them locally, preview changes, and apply them with a Terraform-like lifecycle.

## Install

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
docker run --rm rudderlabs/rudder-cli
```

Mount `~/.rudder` to persist authentication and mount your project directory when running project commands. See the [public documentation](https://www.rudderstack.com/docs/dev-tools/rudder-cli/) for complete Docker usage.

### Build from source

Requires the Go version declared in [`go.mod`](go.mod).

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

## Quickstart

```sh
rudder-cli auth login
mkdir rudder-project && cd rudder-project
rudder-cli import workspace
rudder-cli validate
rudder-cli apply --dry-run
```

## Learn more

- [Rudder CLI documentation](https://www.rudderstack.com/docs/dev-tools/rudder-cli/)
- [Validation rules](https://www.rudderstack.com/docs/dev-tools/rudder-cli/validation-rules/)
- [Breaking changes](BREAKING_CHANGES.md)
- [Contributing](CONTRIBUTING.md)
