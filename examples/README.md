# Rudder CLI examples

Each `examples/<kind>/` directory is a self-contained project for that supported spec kind. Files named `minimal.yaml` show the smallest useful shape; `full.yaml` demonstrates richer configuration, and other sibling specs satisfy local references.

The unit tests in `cli/internal/app/examples_test.go` load every directory through the normal project validation pipeline without credentials or network access. They also require every kind directory to include `minimal.yaml` and `full.yaml`, each containing at least one spec whose `kind` matches the directory. A directory that needs experimental provider registration lists one flag name per line in `.flags`.

Because all YAML files in a kind directory load together, use the following sibling files when copying an individual scenario into another project:

| Kind | `minimal.yaml` needs | `full.yaml` needs |
| --- | --- | --- |
| `account` | None | None |
| `categories` | None | None |
| `custom-types` | None | None |
| `data-graph` | None | None |
| `destination` | None | None |
| `event-stream-connections` | `source.yaml`, `destination.yaml` | `mobile-source.yaml`, `destination.yaml` |
| `event-stream-source` | None | `tracking-plan.yaml`, `events.yaml` |
| `events` | None | `category.yaml` |
| `import-manifest` | `properties.yaml` | `properties.yaml` |
| `properties` | None | None |
| `retl-connections` | `source.yaml`, `account.yaml`, `destination.yaml` | `orders-source.yaml`, `account.yaml`, `full-destination.yaml` |
| `retl-source-sql-model` | None | `account.yaml` |
| `retl-source-table` | None | `account.yaml` |
| `tp` | `basic-events.yaml` | `events.yaml`, `properties.yaml` |
| `tracking-plan` | `events.yaml` | `events.yaml`, `properties.yaml` |
| `transformation` | None | `library.yaml` |
| `transformation-library` | None | None |

To validate the catalog locally:

```sh
go test ./cli/internal/app -run TestExamples -count=1
```
