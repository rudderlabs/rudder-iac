# Cloning a Workspace

`rudder-cli init` writes a whole RudderStack workspace into an empty directory as
YAML specs — everything in it, whether or not the CLI already manages it.

That "whether or not" is the difference from `import workspace`. Import exists to
**add** to a project you already have, so it only ever brings down resources the
CLI does not manage yet; the specs for the managed ones are assumed to be in your
repo already. `init` makes no such assumption. It is the command for the case
where you have credentials for a workspace and nothing else: a fresh machine, a
CI runner, a sandbox, an agent session starting from an empty directory.

```bash
rudder-cli init --location ./my-project
```

Applying an untouched project written by `init` reports **no changes**. That is
the property the command is built around — the managed resources keep the
identifiers they already carry upstream, so nothing reads as a rename, a
re-create, or a delete.

---

## Choosing providers

With no arguments, `init` clones every provider. Naming one or more limits it:

```bash
rudder-cli init eventstream destination --location ./my-project
```

Valid names: `datacatalog`, `retl`, `eventstream`, `transformations`,
`datagraph`, `account`, `destination`. An unknown name fails fast and lists the
valid ones.

A partial clone is a partial project. If you clone `eventstream` without
`destination`, the specs will reference destinations that are not in the project,
and `validate` will say so — so scope the clone only when you mean to work on
that slice.

---

## The target directory

`init` writes specs at the **root** of `--location`, not into an `imported/`
subdirectory: a clone is the whole project, so there is nothing to unpack.

It refuses to run if the directory already contains `.yaml` or `.yml` files:

```
target directory is not empty: ./my-project already contains spec files (events.yaml) —
use 'rudder-cli import workspace' to add remote resources to an existing project
```

Other files — a `README.md`, a `.gitignore`, a var file — are not in the way. A
directory that does not exist yet is created.

The reason for the refusal is duplication. A clone reproduces every resource in
the workspace; written over a project that already describes some of them, you
would end up with two specs per resource, and the next `apply` would fail on the
unique-name collisions. `import workspace` is the command that handles that case,
and `--merge` is the flag that links rather than duplicates.

---

## Secrets

Secrets are not readable from the API, so `init` cannot bring them down. Every
secret field comes out as a variable reference:

```yaml
spec:
  accessKey: {{ .DESTINATION_S3_ACCESS_KEY }}
```

alongside a scaffolded `secrets.vars.yaml` holding a placeholder per variable.
Fill it in, keep it out of version control, and pass it to apply:

```bash
rudder-cli apply --location ./my-project --var-file ./my-project/secrets.vars.yaml
```

An unfilled (null) placeholder makes `apply` fail rather than silently sending an
empty secret. Until the var file is filled in, the specs are not parseable on
their own — the `{{ }}` token is substituted before the YAML is read — so pass
`--var-file` to `validate` too.

---

## Command reference

```
rudder-cli init [provider...] [flags]
```

| Flag               | Description                                   |
| ------------------ | --------------------------------------------- |
| `--location`, `-l` | Directory to write the project into (default `.`). |

---

## `init` vs `import workspace`

|                             | `init`                                  | `import workspace`                     |
| --------------------------- | --------------------------------------- | -------------------------------------- |
| Brings down managed resources | yes                                    | no                                     |
| Target                      | an empty directory                      | an existing project                    |
| Writes to                   | the project root                        | `imported/`                            |
| Requires a synced project   | n/a — there is no project yet           | yes (unless `--merge`)                 |
| Emits an import manifest    | no                                      | with `--merge`, behind `importMerge`    |
| First `apply` after it      | no changes                              | adopts the newly imported resources    |

---

## Limitations

- **No refresh.** `init` clones once, into an empty directory. Bringing an
  existing project back in line with a workspace that drifted is a different
  problem and not something `init` does — re-running it over a written project is
  refused.
- **Secrets are yours to supply** (see [Secrets](#secrets)).
- **Destination types the CLI does not know are skipped.** A destination whose
  `(type, version)` pair has no registered definition cannot be expressed as a
  spec, so it is left out of the clone rather than written wrong.
- **Data graph children that exist only upstream follow the data graph.** A data
  graph is one composite spec; its models and relationships come down with it.
