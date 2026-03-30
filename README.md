# rudder-cli

> **Warning**: This tool is a work in progress. Expect frequent changes and no backwards compatibility guarantees at this stage.

## Installation

### Binary Releases

Download the latest binary for your platform:

**macOS (Apple Silicon)**:
```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

**macOS (Intel)** / **Linux**:
```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Docker

```sh
# Basic usage
docker run rudderlabs/rudder-cli

# With config mounted
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli

# With catalog files and access token
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

### Build from Source

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/

# Or build Docker image
make docker-build
```
