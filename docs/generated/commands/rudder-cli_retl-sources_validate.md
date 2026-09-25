---
command: "rudder-cli retl-sources validate"
---

## rudder-cli retl-sources validate

Validate a RETL source (SQL model or table)

### Synopsis

Validate a RETL source (SQL model or warehouse table) by executing its query without returning data. s3 table sources have no query to validate.

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

* [rudder-cli retl-sources](rudder-cli_retl-sources.md)	 - Manage RETL sources
