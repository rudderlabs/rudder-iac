---
command: rudder-cli retl-sources
---

## rudder-cli retl-sources

Manage RETL sources

### Synopsis

Validate or preview RETL SQL model source specs from a local project before applying them. Select a source by its external ID and use --location when the project is not in the current directory.

### Examples

```
  rudder-cli retl-sources validate orders-model --location ./project
  rudder-cli retl-sources preview orders-model --limit 5
```

### Options

```
  -h, --help   help for retl-sources
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli retl-sources preview](rudder-cli-retl-sources-preview.md)	 - Preview a RETL source SQL model
* [rudder-cli retl-sources validate](rudder-cli-retl-sources-validate.md)	 - Validate a RETL source SQL model

