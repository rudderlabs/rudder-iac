---
command: "rudder-cli apply"
---

## rudder-cli apply

Apply project configuration changes

### Synopsis

Applies the project configuration changes to the RudderStack workspace associated with your access token.
This includes creating, updating, or deleting resources based on
the differences between local configuration and the workspace resources.


```
rudder-cli apply [flags]
```

### Examples

```
rudder-cli apply --location ./project
rudder-cli apply --location ./project --dry-run
rudder-cli apply --location ./project --confirm=false

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
