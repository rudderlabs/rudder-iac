# rudder-cli

Go CLI for [RudderStack](https://www.rudderstack.com/) Infrastructure-as-Code (declarative resources via YAML, Terraform-like apply). **Work in progress** — APIs and behavior may change; backwards compatibility is not guaranteed yet.

## Install

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

**Linux (x86_64)**

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

**Docker**

```sh
docker run rudderlabs/rudder-cli
```

Persist config (`~/.rudder`):

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

Catalog on disk (example):

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

Token via env instead of local config:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

**Build from source** (requires Go)

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

Local Docker image: `make docker-build`.
