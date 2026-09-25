---
command: rudder-cli validate
---

## rudder-cli validate

Validate project configuration

### Synopsis

Validate local RudderStack project files without applying changes.
The command checks strict YAML syntax, required fields, provider rules, resource references,
dependency cycles, and workspace-aware constraints. Use --location for a file or directory and
repeat --var-file to resolve environment-specific values before validation.


```
rudder-cli validate [flags]
```

### Examples

```
$ rudder-cli validate --location </path/to/dir or file>

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

