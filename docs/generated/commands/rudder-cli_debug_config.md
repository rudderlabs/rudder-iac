---
command: "rudder-cli debug config"
---

## rudder-cli debug config

Dump the active configuration

### Synopsis

Print the effective CLI configuration as formatted JSON for troubleshooting.

```
rudder-cli debug config [flags]
```

### Examples

```
RUDDERSTACK_CLI_DEBUG=true rudder-cli debug config

```

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli debug](rudder-cli_debug.md)	 - Debug commands
