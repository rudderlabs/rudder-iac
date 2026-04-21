# rudder-cli

`rudder-cli` is a Go CLI for managing RudderStack resources using Infrastructure-as-Code.

## Status

> **Work in progress:** frequent changes are expected and backwards compatibility is not guaranteed yet.

## Installation

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

### Linux

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

## Docker

Run the CLI:

```sh
docker run --rm rudderlabs/rudder-cli
```

Use local config (`~/.rudder`):

```sh
docker run --rm -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

Run commands with a local catalog:

```sh
docker run --rm -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

Use token from environment:

```sh
docker run --rm -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

## Build from Source

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

Build Docker image locally:

```sh
make docker-build
```
