# AGENTS.md

## Cursor Cloud specific instructions

### Overview

This is a Go CLI tool (`rudder-cli`) for RudderStack Infrastructure-as-Code management. It is a standalone binary that talks to the RudderStack Public API — there are no local databases, message queues, or Docker Compose stacks required for development.

### Quick reference

Standard commands are documented in `CLAUDE.md` and `Makefile`:
- `make build` — build CLI binary to `bin/rudder-cli`
- `make lint` — run golangci-lint (gosec)
- `make test` — unit tests (excludes `cli/tests/`)
- `make test-e2e` — end-to-end tests (requires `RUDDERSTACK_ACCESS_TOKEN`)
- `make test-all` — unit + E2E tests

### Gotchas

- **`go.mod` replace directive**: The file contains `replace github.com/rudderlabs/rudder-data-catalog-provider/sdk => ../rudder-data-catalog-provider/sdk` pointing to a sibling repo that does not exist in the workspace. This is safe to ignore — no Go source files import this module, so builds and tests are unaffected.
- **One unit test requires API credentials**: `cli/pkg/exp/project/project_test.go` (`TestProjectLoad`) fails without `RUDDERSTACK_ACCESS_TOKEN` set. This is an integration-like test that lives outside `cli/tests/`. All other unit tests pass without credentials.
- **E2E tests need real API access**: Tests in `cli/tests/` require `RUDDERSTACK_ACCESS_TOKEN` and optionally `RUDDERSTACK_API_URL`. They interact with a real RudderStack workspace.
- **Most CLI commands require auth**: Commands like `validate`, `apply`, `import` need an access token. Use `RUDDERSTACK_ACCESS_TOKEN` env var or `rudder-cli auth login`. The `typer options` command works without auth and is useful for quick smoke-testing the binary.
- **Go version**: The project requires Go 1.24.0 (specified in `go.mod`). The golangci-lint tool auto-downloads a newer Go toolchain (1.25.x) for itself, which is expected behavior.
