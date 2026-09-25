---
command: rudder-cli transformations
---

## rudder-cli transformations

Manage transformations

### Synopsis

Test transformation specs and inspect the default event payloads used by local test suites. Use --all or --modified to select multiple project transformations.

### Examples

```
$ rudder-cli transformations test my-transformation-id
$ rudder-cli transformations test --all
$ rudder-cli transformations test --modified

```

### Options

```
  -h, --help   help for transformations
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli transformations show-default-events](rudder-cli-transformations-show-default-events.md)	 - Show default test events
* [rudder-cli transformations test](rudder-cli-transformations-test.md)	 - Test transformations

