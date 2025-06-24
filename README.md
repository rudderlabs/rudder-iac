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
  - [🧪 Schema Import from Source](#-schema-import-from-source)
      - [Overview](#overview)
      - [Enabling ImportFromSource Command](#enabling-importfromsource-command)
      - [Authentication Setup](#authentication-setup)
      - [Configuration File Format](#configuration-file-format)
      - [Command Examples](#command-examples)
      - [Generated Output Structure](#generated-output-structure)

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

To run schema import commands with Docker:

```sh
# Import schemas with experimental mode enabled
docker run -v ~/.rudder:/.rudder -v $(pwd):/workspace \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  rudderlabs/rudder-cli tp importFromSource /workspace/output/ --verbose

# Import with access token and specific writeKey
docker run -v $(pwd):/workspace \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  -e RUDDERSTACK_ACCESS_TOKEN="your-access-token" \
  rudderlabs/rudder-cli tp importFromSource /workspace/output/ --write-key "your-write-key" --verbose

# Import with custom event type mappings
docker run -v ~/.rudder:/.rudder -v $(pwd):/workspace \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  rudderlabs/rudder-cli tp importFromSource /workspace/output/ --config /workspace/mappings.yaml --verbose
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

## 🧪 Schema Import from Source

#### Overview

The `importFromSource` command provides an optimized, all-in-one workflow to import RudderStack event schemas from the Event Audit API and convert them into RudderStack Data Catalog YAML files for tracking plan management.

**Optimized Workflow**: Fetch → Unflatten → Convert (all in memory)

#### Enabling ImportFromSource Command

The `importFromSource` command requires experimental mode to be enabled. Enable experimental mode using one of these methods:

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

#### Configuration File Format

The `importFromSource` command uses a YAML configuration file to specify event-type-specific JSONPath mappings for schema processing.

**Configuration File Format (YAML)**:
```yaml
event_mappings:
  identify: "$.traits"           # or "$.context.traits" or "$.properties"
  page: "$.context.traits"       # or "$.traits" or "$.properties"
  screen: "$.properties"         # or "$.traits" or "$.context.traits"
  group: "$.properties"          # or "$.traits" or "$.context.traits"
  alias: "$.traits"              # or "$.context.traits" or "$.properties"
# Note: "track" always uses "$.properties" regardless of config
```

**Default Behavior**:
- Without writeKey: fetches all schemas
- Without config: all event types use "$.properties" 
- Track events always use "$.properties"

#### Command Examples

**Basic Import from Source**:
```bash
# Enable experimental mode (required)
export RUDDERSTACK_CLI_EXPERIMENTAL=true

# Import all schemas with default configuration
rudder-cli tp importFromSource output/

# Import with specific writeKey
rudder-cli tp importFromSource output/ --write-key "your-write-key"

# Import with custom event type mappings  
rudder-cli tp importFromSource output/ --config mappings.yaml

# Dry run with verbose output
rudder-cli tp importFromSource output/ --config mappings.yaml --dry-run --verbose
```

**Complete Workflow**:
```bash
# Enable experimental mode
export RUDDERSTACK_CLI_EXPERIMENTAL=true

# 1. Import schemas from Event Audit API (all-in-one optimized workflow)
rudder-cli tp importFromSource output/ --verbose

# 2. Validate generated tracking plans
rudder-cli tp validate -l output/

# 3. Deploy tracking plans
rudder-cli tp apply -l output/
```

**Advanced Examples**:
```bash
# Import with custom event mappings
cat > mappings.yaml << EOF
event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  screen: "$.properties"
  group: "$.properties"
  alias: "$.traits"
EOF

rudder-cli tp importFromSource output/ --config mappings.yaml --verbose

# Import for specific source with custom mappings
rudder-cli tp importFromSource output/ --write-key "1a2b3c4d5e" --config mappings.yaml --verbose

# Preview what would be generated
rudder-cli tp importFromSource output/ --dry-run --verbose
```

#### Generated Output Structure

The `importFromSource` command generates a complete RudderStack Data Catalog structure:

```
output/
├── events.yaml              # All unique events extracted from eventIdentifier
├── properties.yaml          # All properties with custom type references
├── custom-types.yaml        # Custom object and array type definitions
└── tracking-plans/          # Individual tracking plans grouped by writeKey
    ├── writekey-source1.yaml
    └── writekey-source2.yaml
```

**Key Features**:
- **Optimized In-Memory Processing**: Eliminates temporary files for better performance
- **Event-Type-Specific Processing**: Different JSONPath mappings per event type
- **Automatic Type Generation**: Creates reusable custom types for complex objects
- **Source Grouping**: Organizes tracking plans by writeKey (source)
- **Data Catalog Compatible**: Generated YAML files work directly with `tp` commands

**Integration with Tracking Plans**:
```bash
# After import, validate and deploy
rudder-cli tp validate -l output/
rudder-cli tp apply -l output/ --dry-run
rudder-cli tp apply -l output/
```

**Docker Integration**:
```bash
# Run importFromSource with Docker
docker run -v ~/.rudder:/.rudder -v $(pwd):/workspace \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  rudderlabs/rudder-cli tp importFromSource /workspace/output/ --verbose

# With custom configuration
docker run -v ~/.rudder:/.rudder -v $(pwd):/workspace \
  -e RUDDERSTACK_CLI_EXPERIMENTAL=true \
  rudderlabs/rudder-cli tp importFromSource /workspace/output/ --config /workspace/mappings.yaml --verbose
```


