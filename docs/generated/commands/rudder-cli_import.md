---
command: "rudder-cli import"
---

## rudder-cli import

Import remote resources to local configuration

### Synopsis

Export supported resources from the authenticated workspace into local declarative YAML project files.

### Examples

```
rudder-cli import workspace --location ./project

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
* [rudder-cli import retl-sources](rudder-cli_import_retl-sources.md)	 - Import remote RETL SQL Model to local configuration
* [rudder-cli import workspace](rudder-cli_import_workspace.md)	 - Import workspace resources
