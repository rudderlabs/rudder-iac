---
command: "rudder-cli workspace accounts list"
---

## rudder-cli workspace accounts list

List accounts in the workspace

### Synopsis

List workspace accounts, optionally filtering by category or account type and emitting JSON.

```
rudder-cli workspace accounts list [flags]
```

### Examples

```
rudder-cli workspace accounts list
rudder-cli workspace accounts list --category source --json

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

* [rudder-cli workspace accounts](rudder-cli_workspace_accounts.md)	 - Manage accounts in the workspace
