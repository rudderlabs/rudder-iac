---
command: "rudder-cli workspace info"
---

## rudder-cli workspace info

Show information about the authenticated workspace

### Synopsis

Display identifying and environment information for the workspace associated with the current access token.

```
rudder-cli workspace info [flags]
```

### Examples

```
rudder-cli workspace info
rudder-cli workspace info --json

```

### Options

```
  -h, --help   help for info
      --json   Output workspace information as JSON
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli workspace](rudder-cli_workspace.md)	 - Manage workspace resources
