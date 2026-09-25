---
command: "rudder-cli experimental enable"
---

## rudder-cli experimental enable

Enable an experimental flag

### Synopsis

Enable a specific experimental flag by name

```
rudder-cli experimental enable <flag-name> [flags]
```

### Examples

```
RUDDERSTACK_CLI_EXPERIMENTAL=true rudder-cli experimental enable importMerge

```

### Options

```
  -h, --help   help for enable
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli experimental](rudder-cli_experimental.md)	 - Manage experimental features
