# rudder-cli

`rudder-iac` is the repository for the `rudder-cli` binary.

> **Work in progress:** `rudder-cli` is under active development. Backward compatibility is not guaranteed yet.

## Installation

Download the latest release for your platform and install `rudder-cli`:

| Platform | Archive |
| --- | --- |
| macOS (Apple Silicon) | `rudder-cli_Darwin_arm64.tar.gz` |
| macOS (Intel) | `rudder-cli_Darwin_x86_64.tar.gz` |
| Linux | `rudder-cli_Linux_x86_64.tar.gz` |

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/<archive> | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Docker

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

Example with catalog files and access token:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

### Build from source

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
