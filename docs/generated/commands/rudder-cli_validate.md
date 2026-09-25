---
command: "rudder-cli validate"
---

## rudder-cli validate

Validate project configuration

### Synopsis

Validates the project configuration files for correctness and consistency.
This includes checking for valid syntax, required fields, and relationships
between resources.


```
rudder-cli validate [flags]
```

### Examples

```
rudder-cli validate --location ./project
rudder-cli validate --location ./project --var-file production.vars.yaml

```

### Options

```
  -h, --help                   help for validate
  -l, --location string        Path to the directory containing the project files or a specific file (default ".")
      --var-file stringArray   Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
