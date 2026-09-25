---
command: "rudder-cli"
---

## rudder-cli

Manage RudderStack resources as code

### Synopsis

Manage RudderStack resources with declarative YAML project files.

Use apply, validate, destroy, and import to manage project state; inspect remote
resources with workspace listings; configure authentication and telemetry; and
use debug or experimental tools when needed. Run help for command guidance or
consult the generated command documentation for the complete reference.

```
rudder-cli [flags]
```

### Examples

```
  # Safely validate local declarative YAML before applying changes
  rudder-cli validate --location ./project

  # Preview the changes without modifying the workspace
  rudder-cli apply --location ./project --dry-run

  # Browse available commands and documentation
  rudder-cli help
```

### Options

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
  -h, --help            help for rudder-cli
```

### SEE ALSO

* [rudder-cli apply](rudder-cli_apply.md)	 - Apply project configuration changes
* [rudder-cli auth](rudder-cli_auth.md)	 - Authentication commands
* [rudder-cli completion](rudder-cli_completion.md)	 - Generate shell completion scripts
* [rudder-cli data-graphs](rudder-cli_data-graphs.md)	 - Manage data graphs
* [rudder-cli debug](rudder-cli_debug.md)	 - Debug commands
* [rudder-cli destroy](rudder-cli_destroy.md)	 - Delete all resources from both the upstream system and state
* [rudder-cli experimental](rudder-cli_experimental.md)	 - Manage experimental features
* [rudder-cli help](rudder-cli_help.md)	 - Help about any command
* [rudder-cli import](rudder-cli_import.md)	 - Import remote resources to local configuration
* [rudder-cli retl-sources](rudder-cli_retl-sources.md)	 - Manage RETL sources
* [rudder-cli telemetry](rudder-cli_telemetry.md)	 - Manage telemetry settings
* [rudder-cli transformations](rudder-cli_transformations.md)	 - Manage transformations
* [rudder-cli typer](rudder-cli_typer.md)	 - Generate type-safe tracking code
* [rudder-cli validate](rudder-cli_validate.md)	 - Validate project configuration
* [rudder-cli workspace](rudder-cli_workspace.md)	 - Manage workspace resources
