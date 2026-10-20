# Contributing to RudderStack

Thanks for taking the time and for your help in improving this project!

## Table of contents

- [**RudderStack Contributor Agreement**](#rudderstack-contributor-agreement)
- [**How you can contribute to RudderStack**](#how-you-can-contribute-to-rudderstack)
- [**Local development**](#local-development)
- [**Committing**](#committing)
- [**Getting help**](#getting-help)

## RudderStack Contributor Agreement

To contribute to this project, we need you to sign the [**Contributor License Agreement (“CLA”)**][CLA] for the first commit you make. By agreeing to the [**CLA**][CLA], we can add you to list of approved contributors and review the changes proposed by you.

## How you can contribute to RudderStack

If you come across any issues or bugs, or have any suggestions for improvement, you can navigate to the specific file in the [**repo**](https://github.com/rudderlabs/rudder-repo-template), make the change, and raise a PR.

You can also contribute to any open-source RudderStack project. View our [**GitHub page**](https://github.com/rudderlabs) to see all the different projects.

## Local development

This repository is a Go project. Install [**Go**](https://go.dev/dl/) **1.24** or newer (see `go.mod` for the exact toolchain version).

From the repository root:

| Command | Description |
| ------- | ----------- |
| `make build` | Build the CLI binary to `bin/rudder-cli`. |
| `make lint` | Run [golangci-lint](https://golangci-lint.run/) (v2) on Go sources via `go run`. |
| `make test` | Run unit tests (excludes `cli/tests`). Produces `coverage-unit.out`. |
| `make test-e2e` | Run end-to-end tests under `cli/tests/`. Produces `coverage-e2e.out`. |
| `make test-it` | Run tests tagged with `integrationtest` (`go test -tags integrationtest ./...`). |
| `make test-all` | Run `make test` then `make test-e2e` (does not run integration tests). |

Run `make help` for a sorted list of targets and short descriptions. For a full check before opening a PR, run `make lint` and the test targets relevant to your change.

## Committing

We prefer squash or rebase commits so that all changes from a branch are committed to master as a single commit. All pull requests are squashed when merged, but rebasing prior to merge gives you better control over the commit message.

## Getting help

For any questions, concerns, or queries, you can start by asking a question in our [**Slack**](https://rudderstack.com/join-rudderstack-slack-community/) community.

### We look forward to your feedback on improving this project!


<!----variables---->

[issue]: https://github.com/rudderlabs/rudder-server/issues/new
[CLA]: https://rudderlabs.wufoo.com/forms/rudderlabs-contributor-license-agreement
