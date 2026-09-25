---
command: "rudder-cli completion zsh"
---

## rudder-cli completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

Source the generated script for the current shell or install it in a directory
listed in fpath so new zsh sessions load rudder-cli completions automatically.


```
rudder-cli completion zsh [flags]
```

### Examples

```
# Load completions for the current shell
source <(rudder-cli completion zsh)

# Install completions for future zsh shells
rudder-cli completion zsh > "${fpath[1]}/_rudder-cli"

```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli completion](rudder-cli_completion.md)	 - Generate shell completion scripts
