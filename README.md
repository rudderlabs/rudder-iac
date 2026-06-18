# rudder-iac

`rudder-iac` is the repository for RudderStack Infrastructure as Code tooling.
It ships the `rudder-cli` binary used to manage RudderStack resources from YAML specs.

## Status

> **Work in progress:** this project is under active development and backward compatibility is not guaranteed yet.

## Install `rudder-cli`

Download the latest release binary for your platform:

- **macOS (Apple Silicon)**
  ```sh
  curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
  sudo mv rudder-cli /usr/local/bin/
  ```
- **macOS (Intel)**
  ```sh
  curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_x86_64.tar.gz | tar -xz rudder-cli
  sudo mv rudder-cli /usr/local/bin/
  ```
- **Linux (x86_64)**
  ```sh
  curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
  sudo mv rudder-cli /usr/local/bin/
  ```

## Use with Docker

Run the published image directly:

```sh
docker run rudderlabs/rudder-cli
```

Typical mounted usage (local config + catalog files):

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

Or provide an access token via environment variable:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
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
