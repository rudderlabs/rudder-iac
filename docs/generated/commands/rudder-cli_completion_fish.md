---
command: "rudder-cli completion fish"
---

## rudder-cli completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

The generated script can be sourced for the current session or written to the
fish completions directory so new shells load rudder-cli completions
automatically.


```
rudder-cli completion fish [flags]
```

### Examples

```
# Load completions for the current shell
rudder-cli completion fish | source

# Install completions for future fish shells
rudder-cli completion fish > ~/.config/fish/completions/rudder-cli.fish

```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli completion](rudder-cli_completion.md)	 - Generate shell completion scripts
