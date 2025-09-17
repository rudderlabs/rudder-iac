# Tracking Plan Export Workflow

This document describes how to export tracking plans and their dependencies from RudderStack using the `rudder-cli` tool.

## Overview

The tracking plan export workflow consists of two main commands:

1. **`tp download`** - Downloads tracking plans and dependencies as JSON files
2. **`tp convert`** - Converts JSON files to rudder-cli compatible YAML format

## Prerequisites

- RudderStack account with tracking plans configured
- `rudder-cli` configured with proper authentication

## Quick Start

### 1. Build the CLI

```bash
# Build the rudder-cli binary
go build ./cli/cmd/rudder-cli

# Or use the pre-built binary if available
./bin/rudder-cli --help
```

### 2. Download Tracking Plan Data

```bash
# Download to default directory (./tracking-plans)
./rudder-cli tp download

# Download to custom directory
./rudder-cli tp download --output-dir ./my-export
```

This creates the following structure:
```
./tracking-plans/
└── json/
    ├── tracking-plans.json     # Tracking plans with event relationships
    ├── events.json            # All events
    ├── properties.json        # All properties
    ├── custom-types.json      # All custom types
    └── categories.json        # All categories
```

### 3. Convert to YAML Format

```bash
# Convert using default directories
./rudder-cli tp convert

# Convert with custom directories
./rudder-cli tp convert --input-dir ./my-export --output-dir ./my-yaml
```

This creates the following structure:
```
./tracking-plans/
└── yaml/
    ├── tracking-plans/
    │   └── {plan-name}.yaml
    ├── events/
    │   └── generated_events.yaml
    ├── properties/
    │   └── generated_properties.yaml
    ├── custom-types/
    │   └── generated_custom_types.yaml
    └── categories/
        └── generated_categories.yaml
```

### 4. Validate Generated YAML

```bash
# Validate the converted YAML files
./bin/rudder-cli tp validate -l tracking-plans/yaml
```

## Command Details

### Download Command

```bash
rudder-cli tp download [flags]
```

**Flags:**
- `--output-dir string` - Output directory for downloaded files (default: "./tracking-plans")

**Features:**
- Downloads complete tracking plan metadata with event relationships
- Includes detailed event entities with properties
- Handles pagination for large datasets
- Captures all dependencies (events, properties, custom types, categories)
- Preserves hierarchical relationships between resources

### Convert Command

```bash
rudder-cli tp convert [flags]
```

**Flags:**
- `--input-dir string` - Input directory containing JSON files (default: "./tracking-plans")
- `--output-dir string` - Output directory for YAML files (default: "./tracking-plans")

**Features:**
- Converts JSON to rudder-cli compatible YAML format
- Groups resources into logical files
- Uses proper reference format (`#/events/{group}/{id}`)
- Maintains cross-references between resources
- Generates validation-ready YAML files

## Complete End-to-End Workflow

```bash
# 1. Build the CLI
go build ./cli/cmd/rudder-cli

# 2. Download tracking plan data from RudderStack
./rudder-cli tp download

# 3. Convert JSON to YAML format
./rudder-cli tp convert

# 4. Validate the generated YAML
./bin/rudder-cli tp validate -l tracking-plans/yaml

# 5. (Optional) Use with other rudder-cli commands
./bin/rudder-cli tp apply -l tracking-plans/yaml
```

## YAML Output Format

The generated YAML files follow the rudder-cli resource format:

### Tracking Plan
```yaml
version: rudder/v0.1
kind: tp
metadata:
  name: my_tracking_plan
spec:
  id: tp_123
  display_name: "My Tracking Plan"
  description: "Plan description"
  rules:
    - type: event_rule
      id: event_123_rule
      event:
        $ref: "#/events/generated_events/event_123"
        allow_unplanned: false
      properties:
        - $ref: "#/properties/generated_properties/prop_456"
          required: true
```

### Events
```yaml
version: rudder/v0.1
kind: events
metadata:
  name: generated_events
spec:
  events:
    - id: event_123
      name: "Page View"
      event_type: "track"
      description: "User viewed a page"
      category: "#/categories/generated_categories/cat_789"
```

### Properties
```yaml
version: rudder/v0.1
kind: properties
metadata:
  name: generated_properties
spec:
  properties:
    - id: prop_456
      name: "page_url"
      type: "string"
      description: "URL of the page"
```

## Troubleshooting

### Authentication Issues
Ensure your `rudder-cli` is properly configured:
```bash
./rudder-cli config --help
```

### Validation Errors
If validation fails, check:
1. Reference formats follow `#/{type}/{group}/{id}` pattern
2. All referenced resources exist in their respective files
3. YAML syntax is correct

### Missing Data
If some data is missing:
1. Verify your RudderStack account has access to the resources
2. Check if pagination is working correctly for large datasets
3. Ensure proper authentication and permissions

## Integration with CI/CD

```bash
#!/bin/bash
# Example CI/CD script

# Download latest tracking plans
./rudder-cli tp download --output-dir ./exports

# Convert to YAML
./rudder-cli tp convert --input-dir ./exports --output-dir ./exports

# Validate before deployment
./bin/rudder-cli tp validate -l exports/yaml

# Deploy if validation passes
if [ $? -eq 0 ]; then
    ./bin/rudder-cli tp apply -l exports/yaml
fi
```

This workflow enables you to maintain tracking plans as code and integrate them into your development and deployment processes.