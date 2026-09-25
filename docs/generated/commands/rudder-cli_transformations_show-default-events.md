---
command: "rudder-cli transformations show-default-events"
---

## rudder-cli transformations show-default-events

Show default test events

### Synopsis

Displays the built-in default test events that are used when testing
transformations without custom test inputs.

These default events cover common RudderStack event types (track, identify,
page, screen, group, alias) and can be used as a reference when creating
custom test inputs for your transformations.


```
rudder-cli transformations show-default-events [flags]
```

### Examples

```
# Show all default test events
$ rudder-cli transformations show-default-events

```

### Options

```
  -h, --help   help for show-default-events
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli transformations](rudder-cli_transformations.md)	 - Manage transformations
