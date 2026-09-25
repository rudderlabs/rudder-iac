---
command: "rudder-cli completion"
---

## rudder-cli completion

Generate shell completion scripts

### Synopsis

Generate shell completion scripts for rudder-cli.

Use this command to install tab completion for a supported shell. Each shell
subcommand prints the script to standard output so you can source it for the
current session or redirect it to the location your shell loads at startup.


```
rudder-cli completion <bash|fish|powershell|zsh> [flags]
```

### Examples

```
# Load bash completions for the current shell
source <(rudder-cli completion bash)

# Install zsh completions for future shells
rudder-cli completion zsh > "${fpath[1]}/_rudder-cli"

```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli](rudder-cli.md)	 - Manage RudderStack resources as code
* [rudder-cli completion bash](rudder-cli_completion_bash.md)	 - Generate the autocompletion script for bash
* [rudder-cli completion fish](rudder-cli_completion_fish.md)	 - Generate the autocompletion script for fish
* [rudder-cli completion powershell](rudder-cli_completion_powershell.md)	 - Generate the autocompletion script for powershell
* [rudder-cli completion zsh](rudder-cli_completion_zsh.md)	 - Generate the autocompletion script for zsh
