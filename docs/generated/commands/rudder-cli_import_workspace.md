---
command: "rudder-cli import workspace"
---

## rudder-cli import workspace

Import workspace resources

### Synopsis

Import upstream workspace resources using available providers into configuration files

```
rudder-cli import workspace [flags]
```

### Examples

```
rudder-cli import workspace --location ./project
rudder-cli import workspace --location ./project --var-file prod.vars.yaml

```

### Options

```
  -h, --help                   help for workspace
  -l, --location string        Path to the directory containing the project files (default ".")
      --merge                  Allow import on a diverged project, linking remote resources that match existing local resources (experimental)
      --var-file stringArray   Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli import](rudder-cli_import.md)	 - Import remote resources to local configuration
