---
command: rudder-cli workspace tracking-plans list
---

## rudder-cli workspace tracking-plans list

List tracking plans in the workspace

### Synopsis

List tracking plans from the authenticated workspace, including their remote IDs and names. Results are printed as a table unless --json is supplied.

```
rudder-cli workspace tracking-plans list [flags]
```

### Examples

```
  rudder-cli workspace tracking-plans list
  rudder-cli workspace tracking-plans list --json
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

* [rudder-cli workspace tracking-plans](rudder-cli-workspace-tracking-plans.md)	 - Inspect tracking plans in the workspace

