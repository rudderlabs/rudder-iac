---
command: rudder-cli
---

## rudder-cli

Manage RudderStack resources as code

### Synopsis

Manage RudderStack workspace resources from declarative YAML project files.

Use rudder-cli to validate and apply local infrastructure-as-code specs, preview
changes before updating a workspace, import existing remote resources, inspect
workspace entities, and generate tracking-plan types. Authentication, telemetry,
and experimental settings are stored in the CLI configuration file selected by
--config. Start with auth login, then use validate and apply for the normal
project workflow.

```
rudder-cli [flags]
```

### Examples

```
  # Authenticate, validate a project, and preview its changes
  rudder-cli auth login
  rudder-cli validate --location ./project
  rudder-cli apply --location ./project --dry-run
```

### Options

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
  -h, --help            help for rudder-cli
```

### SEE ALSO

* [rudder-cli apply](rudder-cli-apply.md)	 - Apply project configuration changes
* [rudder-cli auth](rudder-cli-auth.md)	 - Authentication commands
* [rudder-cli data-graphs](rudder-cli-data-graphs.md)	 - Manage data graphs
* [rudder-cli destroy](rudder-cli-destroy.md)	 - Delete all resources from both the upstream system and state
* [rudder-cli import](rudder-cli-import.md)	 - Import remote resources to local configuration
* [rudder-cli retl-sources](rudder-cli-retl-sources.md)	 - Manage RETL sources
* [rudder-cli telemetry](rudder-cli-telemetry.md)	 - Manage telemetry settings
* [rudder-cli transformations](rudder-cli-transformations.md)	 - Manage transformations
* [rudder-cli typer](rudder-cli-typer.md)	 - Generate type-safe tracking code
* [rudder-cli validate](rudder-cli-validate.md)	 - Validate project configuration
* [rudder-cli workspace](rudder-cli-workspace.md)	 - Inspect workspace resources

