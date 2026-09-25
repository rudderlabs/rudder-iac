---
command: rudder-cli workspace retl-sources list
---

## rudder-cli workspace retl-sources list

List RETL sources in the workspace

### Synopsis

List RETL SQL model sources from the authenticated workspace. Results are printed as a table unless --json is supplied.

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

* [rudder-cli workspace retl-sources](rudder-cli-workspace-retl-sources.md)	 - Inspect RETL sources in the workspace

