---
command: rudder-cli transformations test
---

## rudder-cli transformations test

Test transformations

### Synopsis

Tests transformations by executing their code against test input events.

You can test a single transformation by ID, all transformations, or only
modified transformations. Test results show pass/fail status with optional
verbose output showing diffs for failures.


```
rudder-cli transformations test [id] [flags]
```

### Examples

```
# Test a single transformation
$ rudder-cli transformations test my-transformation-id

# Test all transformations
$ rudder-cli transformations test --all

# Test only modified transformations
$ rudder-cli transformations test --modified

# Test with verbose output (shows diffs)
$ rudder-cli transformations test --all --verbose

# Test from a specific project directory
$ rudder-cli transformations test --all -l ./my-project

# Write results to a custom file path
$ rudder-cli transformations test --all -o /tmp/results.json

# Overwrite an existing results file
$ rudder-cli transformations test --all --force

```

### Options

```
      --all               Test all transformations in the project
      --force             Overwrite output file if it already exists
  -h, --help              help for test
  -l, --location string   Path to the directory containing the project files or a specific file (default ".")
      --modified          Test only new or modified transformations
  -o, --output string     Path to write test results JSON file (default: test-results.json)
      --verbose           Show detailed output including diffs for failures
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is '~/.rudder/config.json') (default "~/.rudder/config.json")
```

### SEE ALSO

* [rudder-cli transformations](rudder-cli-transformations.md)	 - Manage transformations

