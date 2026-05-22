# rudder-cli

`rudder-cli` is the CLI for managing RudderStack resources from local specs in the `rudder-iac` repository.

## ⚠️ Work in Progress

> **Warning**
>
> This tool is actively evolving and backward compatibility is not guaranteed yet.

## Installation

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

### Docker

```sh
docker run rudderlabs/rudder-cli
```

Use `~/.rudder` for persisted config/token:

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

Run against local catalog files:

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

Or pass token directly:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```
