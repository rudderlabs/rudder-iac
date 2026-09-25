---
command: rudder-cli import workspace
---

## rudder-cli import workspace

Import workspace resources

### Synopsis

Export manageable resources from the authenticated workspace into an imported/ directory under the local project. The project must match remote state unless experimental --merge is used; secrets are omitted and must be supplied through variable files.

```
rudder-cli import workspace [flags]
```

### Examples

```
$ rudder-cli import workspace --location </path/to/project_dir>
$ rudder-cli import workspace --location </path/to/project_dir> --var-file prod.vars.yaml

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

* [rudder-cli import](rudder-cli-import.md)	 - Import remote resources to local configuration

