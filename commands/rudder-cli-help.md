---
command: rudder-cli help
---

## rudder-cli help

Show help for a command

### Synopsis

Show detailed help for rudder-cli or one of its commands.

Use this command when you want the same information printed by --help but prefer
subcommand syntax, or when you need help for a nested command path. The command
accepts any visible command path and prints its usage, flags, examples, and child
commands without contacting RudderStack APIs.

```
rudder-cli help [command] [flags]
```

### Examples

```
  # Show root help
  rudder-cli help

  # Show help for a nested command
  rudder-cli help import workspace
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code

