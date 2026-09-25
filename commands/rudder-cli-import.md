---
command: rudder-cli import
---

## rudder-cli import

Import remote resources to local configuration

### Synopsis

Export existing RudderStack resources into local YAML specs that can be managed as code. Import a complete synchronized workspace or select a supported resource-specific workflow.

### Examples

```
  rudder-cli import workspace --location ./project
  rudder-cli import retl-sources --local-id orders --remote-id 2abc123
```

### Options

```
  -h, --help   help for import
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli import retl-sources](rudder-cli-import-retl-sources.md)	 - Import remote RETL SQL Model to local configuration
* [rudder-cli import workspace](rudder-cli-import-workspace.md)	 - Import workspace resources

