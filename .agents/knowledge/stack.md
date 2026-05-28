# Stack

> Dependencies, frameworks, tooling.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.

## 2026-05-18 Dependency And Tooling Snapshot
<!-- ticket:RUD-2739 -->
Language + version (`go.mod`): `go 1.24.0`; module `github.com/rudderlabs/rudder-iac`.

Runtime and CLI framework dependencies (`go.mod`, verbatim versions), grouped by purpose:
- Command/config UX: `github.com/spf13/cobra v1.10.1`, `github.com/spf13/viper v1.21.0`, `github.com/AlecAivazis/survey/v2 v2.3.7`, `github.com/MakeNowJust/heredoc/v2 v2.0.1`, `github.com/kyokomi/emoji/v2 v2.2.13`.
- TUI/output and terminal: `github.com/charmbracelet/bubbletea v1.3.10`, `github.com/charmbracelet/bubbles v0.21.0`, `github.com/charmbracelet/lipgloss v1.1.0`, `github.com/briandowns/spinner v1.23.2`, `golang.org/x/term v0.38.0`, `github.com/tidwall/pretty v1.2.1`.
- Data/config processing and validation: `gopkg.in/yaml.v3 v3.0.1`, `github.com/go-viper/mapstructure/v2 v2.4.0`, `github.com/go-playground/universal-translator v0.18.1`, `github.com/go-playground/validator/v10 v10.30.1`, `github.com/tidwall/sjson v1.2.5`, `github.com/tidwall/gjson v1.18.0`, `github.com/google/uuid v1.6.0`, `github.com/samber/lo v1.52.0`.
- Rudder/SDK and transform support: `github.com/rudderlabs/analytics-go/v4 v4.2.2`, `github.com/evanw/esbuild v0.27.2`; local replace: `replace github.com/rudderlabs/rudder-data-catalog-provider/sdk => ../rudder-data-catalog-provider/sdk`.
- Testing: `github.com/stretchr/testify v1.11.1`, `github.com/google/go-cmp v0.7.0`, `github.com/charmbracelet/x/exp/teatest v0.0.0-20251118172736-77d017256798`.

Build/runtime tooling:
- `Makefile`: build target compiles `./cli/cmd/rudder-cli`; linter runner pinned as `GOLANGCI=github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0`; test targets include `test`, `test-e2e`, `test-it`, `test-all`; image target `docker-build` uses `cli/Dockerfile`.
- `cli/Dockerfile`: builder `FROM golang:1.24.9-alpine@sha256:8f8959f38530d159bf71d0b3eb0c547dc61e7959d8225d1599cf762477384923`; runtime `FROM alpine:latest@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412`; entrypoint `"/usr/local/bin/rudder-cli"`.
- CI workflows (`.github/workflows/*.yml` names): `Docker Build and Push`, `lint`, `Release Please`, `goreleaser`, `Semantic pull requests`, `test with code coverage`, `typer swift validate`, `typer typescript validate`.

## DOCS-2434 — Release packaging is GoReleaser-driven
<!-- ticket:DOCS-2434 -->
- Binary distribution should be treated as a release-pipeline concern: `cli/.goreleaser.yaml` defines the artifact matrix, and `.github/workflows/release.yml` is the publishing path that docs need to stay aligned with.
