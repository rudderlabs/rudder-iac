---
command: "rudder-cli data-graphs"
---

## rudder-cli data-graphs

Manage data graphs

### Synopsis

Manage the lifecycle of data graph resources (models and relationships)

### Examples

```
$ rudder-cli data-graphs validate --all
$ rudder-cli data-graphs validate --modified
$ rudder-cli data-graphs validate model my-model-id

```

### Options

```
  -h, --help   help for data-graphs
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli data-graphs validate](rudder-cli_data-graphs_validate.md)	 - Validate data graph resources
