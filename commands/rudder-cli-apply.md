---
command: rudder-cli apply
---

## rudder-cli apply

Apply project configuration changes

### Synopsis

Apply local project specs to the RudderStack workspace associated with the configured access token.
The command validates the project, compares it with remote state, then creates, updates, imports,
or deletes resources in dependency order. Use --dry-run to inspect the plan without changing the
workspace, --confirm=false for non-interactive execution, and repeat --var-file to resolve variables.


```
rudder-cli apply [flags]
```

### Examples

```
$ rudder-cli apply --location </path/to/dir or file>
$ rudder-cli apply --location </path/to/dir or file> --dry-run
$ rudder-cli apply --location </path/to/dir or file> --confirm=false

```

### Options

```
      --confirm                Confirm changes before applying them (default true)
      --dry-run                Only show the changes without applying them
  -h, --help                   help for apply
  -l, --location string        Path to the directory containing the project files or a specific file (default ".")
      --var-file stringArray   Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code

