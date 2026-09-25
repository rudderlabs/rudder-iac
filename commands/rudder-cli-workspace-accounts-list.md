---
command: rudder-cli workspace accounts list
---

## rudder-cli workspace accounts list

List accounts in the workspace

### Synopsis

List workspace accounts, optionally filtered by account category or type. Results are printed as a table unless --json is supplied.

```
rudder-cli workspace accounts list [flags]
```

### Examples

```
  rudder-cli workspace accounts list
  rudder-cli workspace accounts list --type DESTINATION_SNOWFLAKE --json
```

### Options

```
      --category string   Filter by account category
  -h, --help              help for list
      --json              Output as JSON
      --type string       Filter by account type
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli workspace accounts](rudder-cli-workspace-accounts.md)	 - Inspect accounts in the workspace

