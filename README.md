# rudder-iac

`rudder-iac` is the repository for `rudder-cli`, a CLI for managing RudderStack resources as code.

## Work in Progress

`rudder-cli` is under active development. Breaking changes may occur between releases.

## Install

### macOS (Apple Silicon)

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Linux (x86_64)

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Docker

```sh
docker run --rm rudderlabs/rudder-cli
```

With local config and catalog files:

```sh
docker run --rm -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli apply --dry-run -l /catalog
```

Or pass an access token directly:

```sh
docker run --rm -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli apply --dry-run -l /catalog
```

### Build from Source

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

## Basic Usage

```sh
rudder-cli auth login
rudder-cli validate --location ./project
rudder-cli apply --dry-run --location ./project
```

## Contributing

- See `CONTRIBUTING.md` for contribution workflow.
- Run `make lint` and `make test` before opening a PR.
