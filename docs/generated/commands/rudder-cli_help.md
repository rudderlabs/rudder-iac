---
command: "rudder-cli help"
---

## rudder-cli help

Help about any command

### Synopsis

Show detailed help for rudder-cli or one of its subcommands.

Use this command when you need command syntax, available flags, examples, or
the list of child commands from the installed CLI. Passing a command path shows
help for that command; omitting it shows the root command help.


```
rudder-cli help [command] [flags]
```

### Examples

```
rudder-cli help
rudder-cli help apply
rudder-cli help workspace tracking-plans list

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
