---
command: "rudder-cli retl-sources"
---

## rudder-cli retl-sources

Manage RETL sources

### Synopsis

Preview or validate reverse ETL SQL model resources declared in a local project.

### Examples

```
rudder-cli retl-sources validate my-model --location ./project

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
* [rudder-cli retl-sources preview](rudder-cli_retl-sources_preview.md)	 - Preview a RETL source (SQL model or table)
* [rudder-cli retl-sources validate](rudder-cli_retl-sources_validate.md)	 - Validate a RETL source's spec (SQL model or table)
