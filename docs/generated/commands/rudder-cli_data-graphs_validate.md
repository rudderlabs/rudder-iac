---
command: "rudder-cli data-graphs validate"
---

## rudder-cli data-graphs validate

Validate data graph resources

### Synopsis

Validates data graph resources (models and relationships) against the warehouse.

Checks include table existence, column existence, type compatibility, and more.
You can validate all resources, only modified ones, or a specific resource by type and ID.


```
rudder-cli data-graphs validate [type] [id] [flags]
```

### Examples

```
# Validate all resources
$ rudder-cli data-graphs validate --all

# Validate only modified resources
$ rudder-cli data-graphs validate --modified

# Validate a specific model
$ rudder-cli data-graphs validate model my-model-id

# Validate a specific relationship
$ rudder-cli data-graphs validate relationship my-relationship-id

# Output as JSON
$ rudder-cli data-graphs validate --all --json

```

### Options

```
      --all               Validate all data graph resources in the project
  -h, --help              help for validate
  -j, --json              Output results as JSON
  -l, --location string   Path to the directory containing the project files or a specific file (default ".")
      --modified          Validate only new or modified data graph resources
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli data-graphs](rudder-cli_data-graphs.md)	 - Manage data graphs
