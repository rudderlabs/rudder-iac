# rudder-iac (`rudder-cli`)

`rudder-iac` is the repository for `rudder-cli`, a CLI for managing RudderStack resources with Infrastructure-as-Code workflows.

## ⚠️ Work in Progress

> `rudder-cli` is under active development. Backward compatibility is not guaranteed yet.

## Installation

Install one of the prebuilt binaries:

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

### Linux (x86_64)

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

Alternative install paths:

- Docker image: `docker run rudderlabs/rudder-cli`
- Build from source:

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

## Quickstart (First Run)

Run a dry-run apply for a local catalog directory:

```sh
export RUDDERSTACK_ACCESS_TOKEN=your-access-token
rudder-cli tp apply --dry-run -l /path/to/catalog
```

Docker equivalent:

```sh
docker run -e RUDDERSTACK_ACCESS_TOKEN=your-access-token -v /path/to/catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

If you already use `~/.rudder` config locally, mount it instead of passing the token:

```sh
docker run -v ~/.rudder:/.rudder -v /path/to/catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```
