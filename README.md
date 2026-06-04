# rudder-cli

## Table of Contents

- [Installation](#installation)

## Installation

### Release Binary

Download the latest release for your platform from the [releases page](https://github.com/rudderlabs/rudder-iac/releases/latest), or run:

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

**Linux**
```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Docker

Run the CLI directly:
```sh
docker run rudderlabs/rudder-cli
```

Mount your local config directory:
```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

### Build from Source

Requires [Go](https://go.dev/).

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
