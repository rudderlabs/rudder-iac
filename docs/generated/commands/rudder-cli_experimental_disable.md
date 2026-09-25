---
command: "rudder-cli experimental disable"
---

## rudder-cli experimental disable

Disable an experimental flag

### Synopsis

Disable a specific experimental flag by name

```
rudder-cli experimental disable <flag-name> [flags]
```

### Examples

```
RUDDERSTACK_CLI_EXPERIMENTAL=true rudder-cli experimental disable importMerge

```

### Options

```
  -h, --help   help for disable
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli experimental](rudder-cli_experimental.md)	 - Manage experimental features
