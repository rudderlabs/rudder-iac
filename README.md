# rudder-cli

`rudder-iac` is the repository, and `rudder-cli` is the binary it builds and releases.

## Work in Progress

> This CLI is actively evolving. Breaking changes can happen before a stable release.

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

### Linux (x86_64)

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

## Run with Docker

```sh
docker run rudderlabs/rudder-cli
```

Persist local config (`~/.rudder`) in the container:

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

Use local catalog files with `-l /catalog`:

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

If you are not mounting `~/.rudder`, pass the access token explicitly:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

## Build from Source

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

Build a local Docker image:

```sh
make docker-build
```
