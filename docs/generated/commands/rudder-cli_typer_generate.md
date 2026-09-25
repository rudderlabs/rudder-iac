---
command: "rudder-cli typer generate"
---

## rudder-cli typer generate

Generate type-safe code from tracking plan

### Synopsis

Generate type-safe code from a RudderStack tracking plan

```
rudder-cli typer generate [flags]
```

### Examples

```
$ rudder-cli typer generate --tracking-plan-id tracking-plan-id --platform kotlin
$ rudder-cli typer generate --local --location ./project --platform kotlin

```

### Options

```
  -h, --help                      help for generate
      --local                     Generate from local specs instead of the remote workspace (no workspace, apply, auth or network needed)
  -l, --location string           Path to the project directory or spec file (used with --local) (default ".")
      --option stringArray        Platform-specific options in key=value format (use 'rudder-cli typer options --platform <platform>' to see available options)
  -o, --output string             Output directory for generated files (default ".")
      --platform string           Platform to generate code for (kotlin, swift, typescript) (default "kotlin")
      --tracking-plan-id string   Tracking plan ID to generate code from (remote), or local id of the plan in the specs (with --local)
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli typer](rudder-cli_typer.md)	 - Generate type-safe tracking code
