---
command: rudder-cli typer options
---

## rudder-cli typer options

Show available options for a platform

### Synopsis

Show the supported generator options, descriptions, and defaults for a target platform. Pass --platform with kotlin, swift, or typescript.

```
rudder-cli typer options [flags]
```

### Examples

```
  rudder-cli typer options --platform kotlin
  rudder-cli typer options --platform typescript
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

* [rudder-cli typer](rudder-cli-typer.md)	 - Generate type-safe tracking code

