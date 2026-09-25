---
command: rudder-cli retl-sources validate
---

## rudder-cli retl-sources validate

Validate a RETL source SQL model

### Synopsis

Validate a RETL source SQL model by executing the query without returning data

```
rudder-cli retl-sources validate <external-id> [flags]
```

### Examples

```
$ rudder-cli retl-sources validate my-model
$ rudder-cli retl-sources validate my-model --location ./project

```

### Options

```
  -h, --help              help for validate
  -l, --location string   Path to the project directory (default ".")
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli retl-sources](rudder-cli-retl-sources.md)	 - Manage RETL sources

