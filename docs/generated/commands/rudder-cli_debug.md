---
command: "rudder-cli debug"
---

## rudder-cli debug

Debug commands

### Synopsis

Inspect local CLI diagnostic information. Debug commands require debug mode during normal CLI execution.

### Examples

```
RUDDERSTACK_CLI_DEBUG=true rudder-cli debug config

```

### Options

```
  -h, --help   help for debug
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli debug config](rudder-cli_debug_config.md)	 - Dump the active configuration
* [rudder-cli debug config-file](rudder-cli_debug_config-file.md)	 - Print the path to the active configuration file
* [rudder-cli debug stacktrace](rudder-cli_debug_stacktrace.md)	 - Display the last panic stacktrace from the log file
