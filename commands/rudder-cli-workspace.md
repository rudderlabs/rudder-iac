---
command: rudder-cli workspace
---

## rudder-cli workspace

Inspect workspace resources

### Synopsis

Inspect the authenticated RudderStack workspace and list supported remote resource types without changing them. Listing commands use table output by default and support --json for automation.

### Examples

```
  rudder-cli workspace info
  rudder-cli workspace event-stream-sources list --json
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
* [rudder-cli workspace accounts](rudder-cli-workspace-accounts.md)	 - Inspect accounts in the workspace
* [rudder-cli workspace event-stream-sources](rudder-cli-workspace-event-stream-sources.md)	 - Manage event stream sources in the workspace
* [rudder-cli workspace info](rudder-cli-workspace-info.md)	 - Show information about the authenticated workspace
* [rudder-cli workspace retl-sources](rudder-cli-workspace-retl-sources.md)	 - Inspect RETL sources in the workspace
* [rudder-cli workspace tracking-plans](rudder-cli-workspace-tracking-plans.md)	 - Inspect tracking plans in the workspace

