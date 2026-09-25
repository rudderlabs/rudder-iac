---
command: "rudder-cli completion bash"
---

## rudder-cli completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

The generated script requires the bash-completion package. Source the script
for the current shell or install it in your bash completion directory so new
shells load rudder-cli completions automatically.


```
rudder-cli completion bash
```

### Examples

```
# Load completions for the current shell
source <(rudder-cli completion bash)

# Install completions system-wide on Linux
rudder-cli completion bash > /etc/bash_completion.d/rudder-cli

```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli completion](rudder-cli_completion.md)	 - Generate shell completion scripts
