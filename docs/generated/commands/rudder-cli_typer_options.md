---
command: "rudder-cli typer options"
---

## rudder-cli typer options

Show available options for a platform

### Synopsis

List platform-specific code generation options and their default values.

```
rudder-cli typer options [flags]
```

### Examples

```
rudder-cli typer options --platform kotlin

```

### Options

```
  -h, --help              help for options
      --platform string   Platform to show options for (kotlin, swift, typescript)
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli typer](rudder-cli_typer.md)	 - Generate type-safe tracking code
