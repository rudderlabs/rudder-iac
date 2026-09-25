---
command: rudder-cli retl-sources preview
---

## rudder-cli retl-sources preview

Preview a RETL source SQL model

### Synopsis

Preview a RETL source SQL model to see the data structure and sample rows

```
rudder-cli retl-sources preview <external-id> [flags]
```

### Examples

```
$ rudder-cli retl-sources preview my-model
$ rudder-cli retl-sources preview my-model --location ./project --limit 5
$ rudder-cli retl-sources preview my-model --interactive=false
$ rudder-cli retl-sources preview my-model --json

```

### Options

```
  -h, --help              help for preview
      --interactive       Enable interactive table display (default true)
  -j, --json              Output preview rows as JSON
      --limit int         Number of rows to preview (default 10)
  -l, --location string   Path to the project directory (default ".")
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli retl-sources](rudder-cli-retl-sources.md)	 - Manage RETL sources

