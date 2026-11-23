# Dependency graph export (`rudder-cli graph`)

`rudder-cli graph` exports the **resource dependency graph** of a project — the
resources you declare and the references between them. Render it for humans
(Graphviz / Mermaid) to understand blast radius and structure, or emit JSON to
feed other tools.

It reads the same graph the apply cycle builds, so what you see is exactly what
`apply` reasons about.

## The `graph` command

```bash
# Print a Graphviz DOT graph of the current project
rudder-cli graph

# Choose a format explicitly
rudder-cli graph --format dot
rudder-cli graph --format mermaid
rudder-cli graph --format json

# Point at a specific project directory or file
rudder-cli graph ./specs

# Restrict to one resource type
rudder-cli graph --type event
```

Render the human formats with the usual tools:

```bash
rudder-cli graph --format dot | dot -Tsvg > graph.svg
rudder-cli graph --format mermaid   # paste into any Mermaid viewer
```

## JSON output (stable contract)

`--format json` emits a documented, stable shape that other tools can depend on:

```json
{
  "nodes": [{ "urn": "event:signed_up", "type": "event", "id": "signed_up" }],
  "edges": [{ "from": "event:signed_up", "to": "property:user_id" }],
  "cycles": ["event:a → event:b → event:a"]
}
```

- `urn` is `type:id`; `type`/`id` are its parts.
- `edge.from` **depends on** `edge.to`.
- `nodes`/`edges` are always arrays (never `null`) and are sorted for
  deterministic output; `cycles` is omitted when there are none.

## Example use cases

- **Blast-radius review.** Before changing a shared property or model, export the
  graph to see everything that depends on it.
- **Onboarding / mental model.** A picture of sources → connections → destinations,
  or tracking-plan → events → properties, orients new contributors fast.
- **Cycle detection.** Cycles are reported (never hang), surfacing accidental
  circular references.
- **Tooling data source.** The JSON form is the input for editor/graph
  visualizations (e.g. the planned VS Code graph view).

## Notes

- `graph` currently initializes the full app dependencies, so it needs an access
  token (`RUDDERSTACK_ACCESS_TOKEN` or `rudder-cli auth login`) even though the
  export itself is local. Making local commands credential-free is a tracked
  follow-up.

## How it works

- The project is loaded the same way `validate`/`apply` load it, then
  `provider.ResourceGraph()` produces the graph — `graph` only renders it, it does
  not re-implement graph building.
- `--type` filters to nodes of one resource type and drops edges to filtered-out
  nodes.
