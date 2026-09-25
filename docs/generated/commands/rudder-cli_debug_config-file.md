---
command: "rudder-cli debug config-file"
---

## rudder-cli debug config-file

Print the path to the active configuration file

### Synopsis

Print the path of the configuration file loaded by the CLI.

```
rudder-cli debug config-file [flags]
```

### Examples

```
RUDDERSTACK_CLI_DEBUG=true rudder-cli debug config-file

```

### Options

```
  -h, --help   help for config-file
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli debug](rudder-cli_debug.md)	 - Debug commands
