---
command: "rudder-cli completion powershell"
---

## rudder-cli completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for PowerShell.

The generated script can be evaluated for the current session or added to your
PowerShell profile so new sessions load rudder-cli completions automatically.


```
rudder-cli completion powershell [flags]
```

### Examples

```
# Load completions for the current shell
rudder-cli completion powershell | Out-String | Invoke-Expression

```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli completion](rudder-cli_completion.md)	 - Generate shell completion scripts
