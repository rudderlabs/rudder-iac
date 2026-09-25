---
command: rudder-cli destroy
---

## rudder-cli destroy

Delete all resources from both the upstream system and state

### Synopsis

Deletes all resources from both the upstream system and local state.
This operation is destructive and will remove ALL resources managed
by the CLI, regardless of any configuration files.
Use with extreme caution.


```
rudder-cli destroy [flags]
```

### Examples

```
$ rudder-cli destroy
$ rudder-cli destroy --dry-run
$ rudder-cli destroy --confirm=false

```

### Options

```
      --confirm   Confirm before destroying resources (default true)
      --dry-run   Only show the resources that would be destroyed without actually destroying them
  -h, --help      help for destroy
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code

