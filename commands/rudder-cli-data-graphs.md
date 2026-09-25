---
command: rudder-cli data-graphs
---

## rudder-cli data-graphs

Manage data graphs

### Synopsis

Validate data graph models and relationships in a local project against their warehouse account. Use the validate subcommand to check all, modified, or one selected resource.

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
* [rudder-cli data-graphs validate](rudder-cli-data-graphs-validate.md)	 - Validate data graph resources

