# rudder-cli

`rudder-cli` is the RudderStack infrastructure-as-code CLI.

> [!WARNING]
> This project is still evolving and may introduce breaking changes.

## Installation

### macOS

Apple Silicon:
```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

Intel:
```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Linux

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Docker

Run the CLI directly:
```sh
docker run rudderlabs/rudder-cli
```

Mount local config from `~/.rudder`:
```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

Run commands with local catalog files:
```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

If you do not mount config, pass a token explicitly:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

### Build from Source

Build the binary locally (requires Go):

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
