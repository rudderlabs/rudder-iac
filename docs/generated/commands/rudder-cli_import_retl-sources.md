---
command: "rudder-cli import retl-sources"
---

## rudder-cli import retl-sources

Import remote RETL SQL Model to local configuration

### Synopsis

Import a remote RETL SQL Model source into a local YAML configuration file.
This command fetches the remote SQL Model using the provided remote ID,
creates a local YAML configuration with the specified local ID, and embeds
import metadata for tracking.

Optionally, you can specify a separate location for SQL files using --sql-location.
When provided, the SQL content will be saved as a separate .sql file and the
YAML configuration will reference it using the 'file' field instead of inline 'sql'.


```
rudder-cli import retl-sources [flags]
```

### Examples

```
$ rudder-cli import retl-sources --local-id my-model --remote-id abc123
$ rudder-cli import retl-sources -i analytics-model -r def789 -l ./models
$ rudder-cli import retl-sources --local-id analytics-model --remote-id def789 --location ./models --sql-location ./sql
$ rudder-cli import retl-sources -i analytics-model -r def789 -l ./models -s ./sql

```

### Options

```
  -h, --help                  help for retl-sources
  -i, --local-id string       Local identifier for the imported SQL Model (required)
  -l, --location string       Directory where to save the YAML configuration file (default: current directory) (default ".")
  -r, --remote-id string      Remote RETL source ID to import (required)
  -s, --sql-location string   Directory where to save SQL files separately (optional, if not provided SQL will be inline in YAML)
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli import](rudder-cli_import.md)	 - Import remote resources to local configuration
