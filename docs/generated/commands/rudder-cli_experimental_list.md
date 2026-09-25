---
command: "rudder-cli experimental list"
---

## rudder-cli experimental list

List all available experimental flags

### Synopsis

List every available experimental feature flag, its status, and its environment variable.

```
rudder-cli experimental list [flags]
```

### Examples

```
RUDDERSTACK_CLI_EXPERIMENTAL=true rudder-cli experimental list

```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli experimental](rudder-cli_experimental.md)	 - Manage experimental features
