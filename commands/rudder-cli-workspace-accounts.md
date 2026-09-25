---
command: rudder-cli workspace accounts
---

## rudder-cli workspace accounts

Inspect accounts in the workspace

### Synopsis

Inspect account resources available in the authenticated workspace. Use the list subcommand to filter accounts by category or type.

### Examples

```
  rudder-cli workspace accounts list
  rudder-cli workspace accounts list --category destination --json
```

### Options

```
  -h, --help   help for accounts
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli workspace](rudder-cli-workspace.md)	 - Inspect workspace resources
* [rudder-cli workspace accounts list](rudder-cli-workspace-accounts-list.md)	 - List accounts in the workspace

