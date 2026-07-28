# Importing a Workspace

`rudder-cli import workspace` reads the resources that already exist in your
RudderStack workspace and writes them into local spec files, so you can bring an
existing setup under Infrastructure-as-Code management.

By default, import assumes a clean starting point: it refuses to run against a
project that has **pending changes** (a project that has drifted from the
workspace), and it treats every workspace resource as brand new. If a resource
you already have locally also exists in the workspace, a plain import writes a
**second** copy of it — and because most resources must be unique (by name, and
so on), the next `apply` fails on the collision.

`--merge` (Smart Import) solves that.

> **Status: experimental.** `--merge` is behind an experimental flag and is **off
> by default**. Without the flag, `import workspace` behaves exactly as before and
> `--merge` is rejected. See [Enabling `--merge`](#enabling---merge).

---

## Quick start

1. **Enable the feature** (one-time setup):

   ```bash
   export RUDDERSTACK_CLI_EXPERIMENTAL=true      # turn on experimental mode
   export RUDDERSTACK_X_IMPORT_MERGE=true        # turn on Smart Import
   ```

2. **Run import with `--merge`:**

   ```bash
   rudder-cli import workspace --merge --location ./my-project
   ```

3. **Review the result.** Import writes an `imported/` directory inside your
   project containing the new specs and, when merging, an
   `imported/import-manifest.yaml` linking file.

4. **Apply:**

   ```bash
   rudder-cli apply --location ./my-project
   ```

Resources that were **linked** are adopted (not recreated); everything else
applies as usual.

---

## What `--merge` does

For each unmanaged workspace resource **of a supported type** (see
[What gets linked](#what-gets-linked)), `--merge` looks for a resource **you
already have locally** that has the same unique identity (the same name, for most
types). When it finds one, it **links** the two instead of duplicating:

- the local resource is recorded as the owner of that workspace resource, in
  `imported/import-manifest.yaml`;
- **no duplicate spec file is written** for it.

When there is no local match, the resource is imported normally — a new spec file
with a generated name.

Two things worth knowing:

- **Your local spec wins.** Linking never overwrites your files. If a linked
  workspace resource differs from your local copy (say a different description),
  your spec is left as-is and the difference shows up as a normal change the next
  time you run `apply`.
- **A diverged project is allowed** with `--merge` (that's the point), with one
  exception — see [Limitations](#limitations).

---

## What gets linked

Linking is based on each resource's unique identity, not on comparing its full
contents:

| Resource                                 | Linked when these match         |
| ---------------------------------------- | ------------------------------- |
| Category, custom type, tracking plan     | name                            |
| Event                                    | name and event type             |
| Property                                 | name, type, and item types      |
| Event stream source                      | name                            |
| SQL model (RETL)                         | display name and account        |
| Transformation                           | name                            |
| Library                                  | import name                     |
| Data graph, its models and relationships | see [Data graphs](#data-graphs) |

If nothing local matches a workspace resource, it is imported as new rather than
linked.

---

## The import manifest

When `--merge` links resources, it records them in
`imported/import-manifest.yaml`. Each entry maps one of your local resources to
the workspace resource it now owns, grouped by workspace:

```yaml
version: "rudder/v1"
kind: "import-manifest"
metadata:
  name: "import-manifest"
spec:
  workspaces:
    - workspace_id: "<your-workspace-id>"
      resources:
        - urn: "category:checkout"
          remote_id: "<workspace-resource-id>"
        - urn: "event:order-completed"
          remote_id: "<workspace-resource-id>"
```

This file is what `apply` uses to know that these resources should be **adopted**
rather than created. You can keep it in version control alongside the rest of
your project — it becomes a permanent record of what is linked.

The `importMerge` flag gates the manifest on **both** sides:

- **Generation** — the manifest is written by `import` only while the flag is
  enabled.
- **Recognition** — the `import-manifest` kind is a recognized spec kind (parsed,
  validated, and honored by `apply`) only while the flag is enabled.

With the flag **off**, `import-manifest` is an unrecognized kind, and any command
that loads your project — `apply`, `validate`, and `import workspace` itself —
**fails validation** with an error like `'kind' must be one of [...]` as long as
a manifest file is present in the project.

The practical consequence: once you've generated a manifest, keep `importMerge`
enabled. Turning it off doesn't quietly ignore the manifest — it makes your
project fail to load until you either re-enable the flag or remove the manifest
file (which drops all the links it recorded).

---

## Effect on `apply`

After a merge import, `apply` uses the manifest:

- **Linked resources** are adopted — `apply` takes ownership of the existing
  workspace resource instead of creating a new one, so there is no duplicate and
  no unique-name collision.
- **Content differences** between your local spec and the adopted resource are
  applied as an ordinary update.
- **Everything else** (resources with no workspace counterpart, or imported as
  new) applies exactly as it normally would.

`validate` and the other commands read the manifest the same way — with the
`importMerge` flag enabled, they treat it as ordinary import metadata, so no
extra steps are needed on your part. (With the flag off, the manifest kind is
unrecognized and every command that loads the project fails validation — see
[The import manifest](#the-import-manifest).)

---

## Data graphs

A data graph is authored as a single composite spec — one graph, its models, and
their relationships together — and a workspace can hold only **one data graph per
account**. Linking respects that:

- the **data graph** links by account;
- **models** and **relationships** link by name, but only inside a data graph
  that itself linked;
- when the data graph links, your local composite spec is kept as the source of
  truth: the graph and its matched children are recorded in the manifest, and no
  new spec file is written.

**Models or relationships that exist in the workspace but not in your local spec
are not imported.** They are left untouched in the workspace — neither adopted
nor deleted. If you want to bring one under management, add it to your local data
graph spec.

---

## Enabling `--merge`

`--merge` requires the experimental flag named `importMerge`. Two things must be
true: experimental mode must be on **and** the flag must be enabled. Pick
whichever method fits your workflow.

The same flag also gates the import manifest — both its generation during import
and its recognition by `apply`/`validate` (see [The import manifest](#the-import-manifest)).
So keep it enabled not just to run `--merge`, but for as long as `apply` needs to
honor a manifest you've already generated.

### Option A — Environment variables (great for CI)

```bash
export RUDDERSTACK_CLI_EXPERIMENTAL=true      # turn on experimental mode
export RUDDERSTACK_X_IMPORT_MERGE=true        # turn on Smart Import
```

### Option B — Config file (persistent, `~/.rudder/config.json`)

```json
{
  "experimental": true,
  "flags": {
    "importMerge": true
  }
}
```

The top-level `"experimental": true` is required — without it, all experimental
flags are ignored.

### Option C — CLI command

Once experimental mode is on (Option A or B sets `experimental`), toggle the flag
with:

```bash
rudder-cli experimental enable importMerge
```

Running `--merge` without the flag fails fast:

```
--merge requires the "importMerge" experimental flag to be enabled
```

---

## Command reference

```
rudder-cli import workspace [flags]
```

| Flag               | Description                                                                                                                                                                       |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--location`, `-l` | Path to the project directory (default `.`).                                                                                                                                      |
| `--merge`          | Link workspace resources that match existing local resources instead of writing duplicates, and allow import on a diverged project. Requires the `importMerge` experimental flag. |

Import writes into an `imported/` subdirectory of the project and **fails if that
directory already exists** — move or remove a previous `imported/` before
re-running.

---

## Limitations

- **Pending deletions still block a merge import.** If you have deleted a spec
  locally but not yet applied that deletion, `import workspace --merge` fails with
  an error telling you to apply the deletion first, because importing over it
  could bring the deleted resource back.
- **Linking never overwrites.** A linked resource's local spec is never rewritten
  from the workspace; content differences are resolved at `apply`.
- **Ambiguous matches stop the import.** If two workspace resources would link to
  the same local resource (for example, duplicate data upstream), the import
  fails rather than guessing which one to link.
- **Data graph children that exist only in the workspace are skipped** (see
  [Data graphs](#data-graphs)).
