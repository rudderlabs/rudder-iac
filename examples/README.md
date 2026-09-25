# Rudder CLI examples

Each `examples/<kind>/` directory is a self-contained project for that supported spec kind. Files named `minimal.yaml` show the smallest useful shape; `full.yaml` demonstrates richer configuration, and other sibling specs satisfy local references.

The unit tests in `cli/internal/app/examples_test.go` load every directory through the normal project validation pipeline without credentials or network access. They also require every kind directory to include `minimal.yaml` and `full.yaml`, each containing at least one spec whose `kind` matches the directory. A directory that needs experimental provider registration lists one flag name per line in `.flags`.

To validate the catalog locally:

```sh
go test ./cli/internal/app -run TestExamples -count=1
```
