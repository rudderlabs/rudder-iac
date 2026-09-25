---
command: rudder-cli telemetry
---

## rudder-cli telemetry

Manage telemetry settings

### Synopsis

Manage telemetry settings for the CLI.

Telemetry helps us understand how the CLI is being used and helps us improve it.
No sensitive information is collected. The data collected includes:
- Command usage statistics
- Error occurrences (without sensitive details)
- Basic system information

Use 'status' to check current telemetry settings.
Use 'enable' or 'disable' to modify telemetry collection.


### Examples

```
  rudder-cli telemetry status
  rudder-cli telemetry disable
```

### Options

```
  -h, --help   help for telemetry
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli telemetry disable](rudder-cli-telemetry-disable.md)	 - Disable telemetry
* [rudder-cli telemetry enable](rudder-cli-telemetry-enable.md)	 - Enable telemetry
* [rudder-cli telemetry status](rudder-cli-telemetry-status.md)	 - Show current telemetry status

