---
command: "rudder-cli experimental"
---

## rudder-cli experimental

Manage experimental features

### Synopsis

List and manage opt-in experimental feature flags. Experimental commands require experimental mode during normal CLI execution.

### Examples

```
RUDDERSTACK_CLI_EXPERIMENTAL=true rudder-cli experimental list

```

### Options

```
  -h, --help   help for experimental
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli experimental disable](rudder-cli_experimental_disable.md)	 - Disable an experimental flag
* [rudder-cli experimental enable](rudder-cli_experimental_enable.md)	 - Enable an experimental flag
* [rudder-cli experimental list](rudder-cli_experimental_list.md)	 - List all available experimental flags
* [rudder-cli experimental reset](rudder-cli_experimental_reset.md)	 - Reset all experimental flags to their defaults
