---
command: "rudder-cli debug stacktrace"
---

## rudder-cli debug stacktrace

Display the last panic stacktrace from the log file

### Synopsis

Read the CLI log and display the most recent recorded panic stack trace.

```
rudder-cli debug stacktrace [flags]
```

### Examples

```
RUDDERSTACK_CLI_DEBUG=true rudder-cli debug stacktrace

```

### Options

```
  -h, --help   help for stacktrace
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli debug](rudder-cli_debug.md)	 - Debug commands
