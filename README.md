# rudder-cli

> **⚠️ Work in Progress**: This tool is under active development. Frequent changes and updates may occur. Backwards compatibility is not guaranteed at this stage.

## Installation

### Binary Releases

Download and install from [latest release](https://github.com/rudderlabs/rudder-iac/releases/latest):

**MacOS (Apple Silicon)**
```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

**MacOS (Intel)**
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

```sh
docker run rudderlabs/rudder-cli
```

With local config and catalog files:
```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=<token> rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

### Build from Source

Requires Go. Run:
```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

To build Docker image:
```sh
make docker-build
```
