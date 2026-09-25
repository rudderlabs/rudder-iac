---
command: "rudder-cli experimental reset"
---

## rudder-cli experimental reset

Reset all experimental flags to their defaults

### Synopsis

Reset all experimental flags by removing the experimental section from the configuration

```
rudder-cli experimental reset [flags]
```

### Examples

```
RUDDERSTACK_CLI_EXPERIMENTAL=true rudder-cli experimental reset

```

### Options

```
  -h, --help   help for reset
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli experimental](rudder-cli_experimental.md)	 - Manage experimental features
