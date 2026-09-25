---
command: "rudder-cli retl-sources validate"
---

## rudder-cli retl-sources validate

Validate a RETL source's spec (SQL model or table)

### Synopsis

Validate a RETL source's spec.

This checks the project's specs and that the source is defined in it. It
does not run the source's query: reading from the warehouse is what
`rudder-cli retl-sources preview` is for, and it is opt-in because it
executes a query against live data.


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
