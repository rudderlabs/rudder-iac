---
command: "rudder-cli workspace"
---

## rudder-cli workspace

Manage workspace resources

### Synopsis

Inspect the authenticated RudderStack workspace and list supported remote resource types without changing them.

### Examples

```
rudder-cli workspace info
rudder-cli workspace tracking-plans list --json

```

### Options

```
  -h, --help   help for workspace
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli workspace accounts](rudder-cli_workspace_accounts.md)	 - Manage accounts in the workspace
* [rudder-cli workspace event-stream-sources](rudder-cli_workspace_event-stream-sources.md)	 - Manage event stream sources in the workspace
* [rudder-cli workspace info](rudder-cli_workspace_info.md)	 - Show information about the authenticated workspace
* [rudder-cli workspace retl-sources](rudder-cli_workspace_retl-sources.md)	 - Manage RETL sources in the workspace
* [rudder-cli workspace tracking-plans](rudder-cli_workspace_tracking-plans.md)	 - Manage tracking plans in the workspace
