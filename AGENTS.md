# AGENTS.md

## Cursor Cloud specific instructions

### Overview

This is a Go CLI project (`rudder-cli`) with no local service dependencies. All remote API calls go to `api.rudderstack.com`. Build, lint, and test commands are in the `Makefile`.

### Quick reference

| Task | Command |
|------|---------|
| Build | `make build` |
| Lint | `make lint` |
| Unit tests | `make test` |
| E2E tests | `make test-e2e` |
| Integration tests | `make test-it` |
| All tests | `make test-all` |

See `CLAUDE.md` for full project architecture, code standards, and workflow details.

### Non-obvious caveats

- **`go.mod` replace directive**: The `go.mod` has a `replace` pointing to `../rudder-data-catalog-provider/sdk`. No Go source files import this module, so it does not affect `go build`, `go test`, or `go mod download`. If it ever becomes an active import, the sibling repo would need to be cloned at `../rudder-data-catalog-provider/`.
- **One test requires `RUDDERSTACK_ACCESS_TOKEN`**: `cli/pkg/exp/project/project_test.go` (`TestProjectLoad`) fails without a valid access token. This is the only unit test that needs external auth; all other unit tests pass without credentials.
- **golangci-lint auto-downloads Go toolchain**: The linter (`v2.9.0`) requires Go >= 1.25.0 and will auto-download `go1.25.8` via the Go toolchain mechanism on first run. This is expected behavior.
- **CLI binary location**: `make build` produces `bin/rudder-cli` in the workspace root.
- **No Docker or local services needed**: This is a pure CLI client. No databases, containers, or docker-compose services are required for development.
