# rudder-cli

## Table of Contents

- [rudder-cli](#rudder-cli)
  - [Table of Contents](#table-of-contents)
  - [⚠️ Work in Progress](#️-work-in-progress)
  - [Installation](#installation)
    - [MacOS](#macos)
      - [Apple Silicon](#apple-silicon)
      - [Intel-based](#intel-based)
    - [Linux](#linux)
    - [Docker](#docker)
    - [Build from Source](#build-from-source)
  - [🧪 Schema Management](#-schema-management)
      - [Overview](#overview)
      - [Enabling Schema Commands](#enabling-schema-commands)
      - [Authentication Setup](#authentication-setup)
      - [Available Commands](#available-commands)
      - [Complete Workflow](#complete-workflow)
      - [Command Examples](#command-examples)

## ⚠️ Work in Progress

> **Warning**
>
> Please note that this tool is currently a work in progress. We are actively developing and improving it, and as such, there may be frequent changes and updates. We do not guarantee backwards compatibility at this stage.

## Installation

### MacOS

#### Apple Silicon

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/latest/download/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

#### Intel-based

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

You can run the CLI directly using Docker:

```sh
docker run rudderlabs/rudder-cli
```

If you need to persist your configuration, or provide an external configuration file, you can mount your local config directory into the container. Assuming your config directory is located at `~/.rudder`, you can run the following command:

```sh
docker run -v ~/.rudder:/.rudder rudderlabs/rudder-cli
```

To run commands with local catalog files, mount your files directory and use the `-l` flag. For example:

```sh
docker run -v ~/.rudder:/.rudder -v ~/my-catalog:/catalog rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

This will use the access token from your local configuration file, and the catalog files from the `/catalog` directory. Alternatively, you can use the `RUDDERSTACK_ACCESS_TOKEN` environment variable to provide the access token:

```sh
docker run -v ~/my-catalog:/catalog -e RUDDERSTACK_ACCESS_TOKEN=your-access-token rudderlabs/rudder-cli tp apply --dry-run -l /catalog
```

**Schema Features with Docker**:

To run schema commands with Docker:

```sh
# Convert schemas with experimental mode enabled
docker run -v ~/.rudder:/.rudder -v $(pwd):/workspace \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  rudderlabs/rudder-cli schema convert /workspace/schemas.json /workspace/output/

# Fetch schemas with access token
docker run -v $(pwd):/workspace \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  -e RUDDERSTACK_ACCESS_TOKEN="your-access-token" \
  rudderlabs/rudder-cli schema fetch /workspace/schemas.json
```

### Build from Source

To build the `rudder-cli` from source, you need to have Go installed. Then, run the following commands:

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```

To build the Docker image locally:

```sh
make docker-build
```

## 🧪 Schema Management

#### Overview

The schema management functionality allows you to work with RudderStack event schemas from the Event Audit API. You can fetch schemas, process them, and convert them into RudderStack Data Catalog YAML files for tracking plan management.

**Workflow**: Fetch → Unflatten → Convert → Deploy

#### Enabling Schema Commands

Schema commands require experimental mode to be enabled (legacy requirement). Enable experimental mode using one of these methods:

**Environment Variable** (Recommended):
```bash
export RUDDERSTACK_CLI_EXPERIMENTAL=true
```

**Config File**:
Edit your config file (default: `~/.rudder/config.json`) and set:
```json
{
  "experimental": true
}
```

#### Authentication Setup

Schema commands use the main CLI's authentication system:

1. **Login via CLI** (Recommended):
   ```bash
   rudder-cli auth login
   ```

2. **Environment Variable**:
   ```bash
   export RUDDERSTACK_ACCESS_TOKEN="your-access-token"
   ```

3. **Optional API URL** (defaults to RudderStack API):
   ```bash
   export RUDDERSTACK_API_URL="https://api.rudderstack.com"
   ```

#### Available Commands

| Command | Description |
|---------|-------------|
| `fetch` | Fetch event schemas from the Event Audit API |
| `unflatten` | Convert flattened schema keys to nested JSON structures |
| `convert` | Convert schemas to RudderStack Data Catalog YAML files |

#### Complete Workflow

```bash
# Enable experimental mode
export RUDDERSTACK_CLI_EXPERIMENTAL=true

# 1. Fetch schemas from Event Audit API
rudder-cli schema fetch schemas.json --verbose

# 2. Unflatten dot-notation keys to nested structures
rudder-cli schema unflatten schemas.json unflattened.json --verbose

# 3. Convert to Data Catalog YAML files
rudder-cli schema convert unflattened.json output/ --verbose

# 4. Validate generated tracking plans
rudder-cli tp validate -l output/

# 5. Deploy tracking plans
rudder-cli tp apply -l output/
```

#### Command Examples

**Fetch Schemas**:
```bash
# Fetch all schemas
rudder-cli schema fetch schemas.json

# Fetch schemas for specific writeKey
rudder-cli schema fetch schemas.json --write-key=YOUR_WRITE_KEY

# Dry run to preview what would be fetched
rudder-cli schema fetch schemas.json --dry-run --verbose
```

**Unflatten Schemas**:
```bash
# Unflatten flattened schema keys
rudder-cli schema unflatten input.json output.json

# With verbose output and custom indentation
rudder-cli schema unflatten input.json output.json --verbose --indent 4
```

**Convert to YAML**:
```bash
# Convert schemas to Data Catalog YAML files
rudder-cli schema convert schemas.json output/

# Dry run to preview generated files
rudder-cli schema convert schemas.json output/ --dry-run --verbose

# Custom YAML indentation
rudder-cli schema convert schemas.json output/ --indent 4
```

**Generated Output Structure**:
```
output/
├── events.yaml              # All unique events extracted from eventIdentifier
├── properties.yaml          # All properties with custom type references
├── custom-types.yaml        # Custom object and array type definitions
└── tracking-plans/          # Individual tracking plans grouped by writeKey
    ├── writekey-source1.yaml
    └── writekey-source2.yaml
```

**Integration with Tracking Plans**:
```bash
# After conversion, validate and deploy
rudder-cli tp validate -l output/
rudder-cli tp apply -l output/ --dry-run
rudder-cli tp apply -l output/
```


