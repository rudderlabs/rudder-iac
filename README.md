# rudder-cli

`rudder-cli` is a Go CLI for managing RudderStack resources with Infrastructure as Code.

## Status

> **Work in progress:** changes are frequent and backward compatibility is not guaranteed yet.

## Install

Download the latest release for your OS/architecture and move it into your `PATH`:

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/<artifact>.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

Common artifacts include:
- `rudder-cli_Darwin_arm64`
- `rudder-cli_Darwin_x86_64`
- `rudder-cli_Linux_x86_64`

## Docker

Run the CLI directly:

```sh
docker run --rm rudderlabs/rudder-cli
```

Run against a local catalog:

```sh
docker run --rm \
  -v ~/.rudder:/.rudder \
  -v ~/my-catalog:/catalog \
  rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

Use an access token from env if needed:

```sh
docker run --rm \
  -v ~/my-catalog:/catalog \
  -e RUDDERSTACK_ACCESS_TOKEN=your-access-token \
  rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

## Build from source

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

Build the Docker image locally:

```sh
make docker-build
```
