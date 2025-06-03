# End-to-End Tests for Rudder CLI

This directory contains end-to-end tests for the `rudder-cli` tracking plan (tp) commands.

## Requirements

The tests require the `RUDDERSTACK_ACCESS_TOKEN` environment variable to be set with a valid token to interact with a RudderStack workspace.

```bash
export RUDDERSTACK_ACCESS_TOKEN="your_token"
make test-e2e
```

## Running Tests

The E2E tests can be run using the Makefile target:

```bash
make test-e2e
```

This will execute all tests in the `tests/` directory.

To run a specific test function, use the `TEST` variable:

```bash
make test-e2e TEST=TestTrackingPlan_CreateFlow
```

## Test Data

The tests use data files located within the `tests/tp/` directory by default. This directory is structured to support different test flows:

- `tests/tp/create`: Contains the initial tracking plan definition used by `TestTrackingPlan_CreateFlow`.
- `tests/tp/update`: Contains the updated tracking plan definition used by `TestTrackingPlan_UpdateFlow`.
- `tests/tp/delete`: Contains the minimal definition required to delete resources, used by `TestTrackingPlan_DeleteFlow`.

### Custom Test Data Location

You can specify a different root directory for test data using the `-testdata.root` flag with the `TEST` variable (ensure you include the test name or use `./...`):

```bash
make test-e2e TEST="-testdata.root=path/to/your/custom/data ./..."
```

Replace `path/to/your/custom/data` with the path to your alternative test data root.

## Test Structure and Helpers

Each test follows this structure:
- Fetches the resource state before any CLI commands are run.
- Runs the CLI commands (`validate`, `apply_dry_run`, `apply`) using shared helpers from `tests/utils`.
- Fetches the resource state after the `apply` command.
- Compares the before and after states to validate the effect of the CLI operations.

YAML files are used as input for the CLI, but state validation is performed by comparing the system state before and after the test steps.

```bash
export RUDDERSTACK_ACCESS_TOKEN="your_token"
make test-e2e
``` 