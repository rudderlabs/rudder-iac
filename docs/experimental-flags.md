# Experimental Flags Developer Guidelines

This document provides practical guidelines for developers working with experimental flags in rudder-cli.

## Overview

Experimental flags allow safe rollout of new features by keeping them disabled by default and requiring explicit opt-in. This enables:

- Testing new features without affecting stable functionality
- Gradual feature rollouts
- Safe feature development alongside stable code

## Adding a New Experimental Flag

Add your flag to `cli/internal/config/experimental.go`:

```go
type ExperimentalConfig struct {
    // StatelessCLI enables stateless CLI mode, which does not depend on resource state being persisted across runs
    StatelessCLI  bool `mapstructure:"statelessCLI"`

    // ConcurrentSyncs enables concurrent sync operations when applying changes
    ConcurrentSyncs bool `mapstructure:"concurrentSyncs"`

}
```

That's it! No other files need modification.

## Using Experimental Flags in Code

### Basic Flag Check

```go
package mycommand

import "github.com/rudderlabs/rudder-iac/cli/internal/config"

func someFunction() {
    cfg := config.GetConfig()

    if cfg.ExperimentalFlags.YourNewFeature {
        // New experimental behavior
        return useNewImplementation()
    } else {
        // Existing stable behavior
        return useStableImplementation()
    }
}
```

### Conditional Command Registration

```go
func init() {
    rootCmd.AddCommand(standardCmd)

    cfg := config.GetConfig()
    if cfg.ExperimentalFlags.ExperimentalCommand {
        rootCmd.AddCommand(experimentalCmd)
    }
}
```

### Enhanced Behavior Pattern

```go
cfg := config.GetConfig()
processor := NewProcessor()

if cfg.ExperimentalFlags.EnhancedProcessing {
    processor.EnableExperimentalFeatures()
}

return processor.Process(data)
```

## User Interface

Users can manage experimental flags through CLI commands:

```bash
# List all experimental flags and their status
rudder-cli experimental list

# Enable a flag
rudder-cli experimental enable statelessCLI

# Disable a flag
rudder-cli experimental disable concurrentSyncs

# Reset all flags to default (false)
rudder-cli experimental reset
```

## Configuration File

Flags are stored in `~/.rudder/config.json`:

```json
{
  "experimental": true,
  "flags": {
    "statelessCLI": true,
    "concurrentSyncs": false
  }
}
```

In order for them to be effective, the top-level `experimental` field must also be set to `true`.

## Environment Variables

Flags can be set via environment variables using the `RUDDERSTACK_X_` prefix:

```bash
export RUDDERSTACK_X_STATELESS_CLI=true
export RUDDERSTACK_X_CONCURRENT_SYNCS=true
rudder-cli apply
```

Environmental variables are automatically binded to viper config, by looking at mapstructure tags of the `ExperimentalConfig` struct fields. They are converted to UPPER_SNAKE_CASE and prefixed with `RUDDERSTACK_X`.

## Best Practices

### Safety First

- **Default to `false`**: All experimental flags must default to disabled
- **Graceful degradation**: Experimental features should never break core functionality
- **Stable fallback**: Always provide stable behavior when flags are disabled

### Code Patterns

- **Clear conditionals**: Use explicit if/else blocks for flag checks
- **Single responsibility**: Each flag should control a single, well-defined feature
- **Consistent naming**: Use camelCase for struct fields, mapstructure names for CLI commands and config.

### Development Workflow

1. **Develop behind flag**: Build new features with the flag disabled by default
2. **Test both paths**: Ensure both experimental and stable code paths work
3. **Document behavior**: Comment why the flag exists and what it enables
4. **Monitor usage**: Telemetry automatically tracks flag usage

## Telemetry

Experimental flag status is automatically included in all command telemetry. No additional code is required - the `telemetry.TrackCommand()` function automatically serializes the experimental config and includes it in analytics.

## Naming Conventions

### Struct Fields (camelCase)

```go
StatelessCLI    bool `mapstructure:"statelessCLI"`
ConcurrentSyncs bool `mapstructure:"concurrentSyncs"`
```

### Environment Variables (UPPER_SNAKE_CASE)

```bash
RUDDERSTACK_X_STATELESS_CLI=true
RUDDERSTACK_X_CONCURRENT_SYNCS=true
```

## Common Pitfalls

❌ **Don't modify multiple files** - Only edit `experimental.go`  
❌ **Don't default to `true`** - All flags must default to `false` for safety  
❌ **Don't break stable behavior** - Experimental features should be additive

## Flag Lifecycle

1. **Experimental**: New feature, disabled by default, requires explicit opt-in
2. **Stable**: Feature proven, can be promoted to standard functionality
3. **Deprecated**: Feature being phased out, flag can be removed

When promoting experimental features to stable:

1. Make the feature always enabled
2. Remove the flag check from code
3. Remove the flag from `ExperimentalConfig`
4. Update documentation
