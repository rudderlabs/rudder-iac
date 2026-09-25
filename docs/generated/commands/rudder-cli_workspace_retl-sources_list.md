---
command: "rudder-cli workspace retl-sources list"
---

## rudder-cli workspace retl-sources list

List RETL sources in the workspace

### Synopsis

List reverse ETL sources in the authenticated workspace, as a table or JSON.

```
rudder-cli workspace retl-sources list [flags]
```

### Examples

```
rudder-cli workspace retl-sources list
rudder-cli workspace retl-sources list --json

```

### Options

```
  -h, --help   help for list
      --json   Output as JSON
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli workspace retl-sources](rudder-cli_workspace_retl-sources.md)	 - Manage RETL sources in the workspace
