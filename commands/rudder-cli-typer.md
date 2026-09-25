---
command: rudder-cli typer
---

## rudder-cli typer

Generate type-safe tracking code

### Synopsis

Generate type-safe tracking code from a remote or local RudderStack tracking plan. Inspect platform defaults with typer options, then use typer generate to write Kotlin, Swift, or TypeScript output.

### Examples

```
  rudder-cli typer options --platform kotlin
  rudder-cli typer generate --tracking-plan-id 2abc123 --platform kotlin
```

### Options

```
  -h, --help   help for typer
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli typer generate](rudder-cli-typer-generate.md)	 - Generate type-safe code from tracking plan
* [rudder-cli typer options](rudder-cli-typer-options.md)	 - Show available options for a platform

