---
command: rudder-cli workspace event-stream-sources list
---

## rudder-cli workspace event-stream-sources list

List event stream sources in the workspace

### Synopsis

List event stream sources from the authenticated workspace, including their IDs, names, types, and enabled state. Use --json for machine-readable output.

```
rudder-cli workspace event-stream-sources list [flags]
```

### Examples

```
  rudder-cli workspace event-stream-sources list
  rudder-cli workspace event-stream-sources list --json
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

* [rudder-cli workspace event-stream-sources](rudder-cli-workspace-event-stream-sources.md)	 - Manage event stream sources in the workspace

