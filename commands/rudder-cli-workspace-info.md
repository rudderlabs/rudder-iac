---
command: rudder-cli workspace info
---

## rudder-cli workspace info

Show information about the authenticated workspace

### Synopsis

Fetch and display identity, environment, status, region, and data-plane details for the workspace associated with the configured access token. Use --json for machine-readable output.

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

* [rudder-cli workspace](rudder-cli-workspace.md)	 - Inspect workspace resources

