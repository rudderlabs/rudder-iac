# Data Graph - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

**No validation rules exist** in the new validation engine for data graph resources (any version). The data graph provider does not implement `RuleHandler`, so `SyntacticRules()` and `SemanticRules()` return empty slices.

All existing validations live in the old handler-based framework:
- `cli/internal/providers/datagraph/handlers/datagraph/handler.go` (ValidateResource)
- `cli/internal/providers/datagraph/handlers/model/handler.go` (ValidateResource)
- `cli/internal/providers/datagraph/handlers/relationship/handler.go` (ValidateResource)

---

## Structural Differences: V0.1 vs V1

Data graph uses a single spec kind (`data-graph`) with inline models and relationships. There is **no structural difference** between V0.1 and V1 -- the same structs are used.

### Structs (`model/datagraph.go`)

```go
type DataGraphSpec struct {
    ID        string      `json:"id" mapstructure:"id"`
    AccountID string      `json:"account_id" mapstructure:"account_id"`
    Models    []ModelSpec `json:"models,omitempty" mapstructure:"models"`
}

type ModelSpec struct {
    ID            string             `json:"id" mapstructure:"id"`
    DisplayName   string             `json:"display_name" mapstructure:"display_name"`
    Type          string             `json:"type" mapstructure:"type"`       // "entity" or "event"
    Table         string             `json:"table" mapstructure:"table"`
    Description   string             `json:"description,omitempty" mapstructure:"description"`
    Relationships []RelationshipSpec `json:"relationships,omitempty" mapstructure:"relationships"`
    PrimaryID     string             `json:"primary_id,omitempty" mapstructure:"primary_id"`    // entity only
    Root          bool               `json:"root,omitempty" mapstructure:"root"`                // entity only
    Timestamp     string             `json:"timestamp,omitempty" mapstructure:"timestamp"`      // event only
}

type RelationshipSpec struct {
    ID            string `json:"id" mapstructure:"id"`
    DisplayName   string `json:"display_name" mapstructure:"display_name"`
    Cardinality   string `json:"cardinality" mapstructure:"cardinality"`     // one-to-one, one-to-many, many-to-one
    Target        string `json:"target" mapstructure:"target"`               // #data-graph-model:<id>
    SourceJoinKey string `json:"source_join_key" mapstructure:"source_join_key"`
    TargetJoinKey string `json:"target_join_key" mapstructure:"target_join_key"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("data-graph", "rudder/v1")` and decode into `DataGraphSpec`.

### Data Graph Level

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `account_id` required | Data graph must have a non-empty `account_id` field |

### Model Level (inline within data graph)

| # | Validation | Description |
|---|-----------|-------------|
| 2 | `display_name` required | Each model must have a non-empty `display_name` |
| 3 | `type` in `entity`, `event` | Model type must be one of: `entity`, `event` |
| 4 | `table` required | Each model must have a non-empty `table` field |
| 5 | Entity model: `primary_id` required | When `type` is `entity`, `primary_id` must be non-empty |
| 6 | Event model: `timestamp` required | When `type` is `event`, `timestamp` must be non-empty |

### Relationship Level (inline within model)

| # | Validation | Description |
|---|-----------|-------------|
| 7 | `display_name` required | Each relationship must have a non-empty `display_name` |
| 8 | `cardinality` required | Each relationship must have a non-empty `cardinality` field |
| 9 | `target` required | Each relationship must reference a target model (format `#data-graph-model:<id>`) |
| 10 | `source_join_key` required | Each relationship must have a non-empty `source_join_key` |
| 11 | `target_join_key` required | Each relationship must have a non-empty `target_join_key` |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Model: data graph URN exists | Each model's parent data graph must exist in the resource graph |
| 2 | Relationship: data graph, source model, target model URNs exist | All referenced resources must exist in the graph |
| 3 | Relationship: event-to-event forbidden | Relationships between two event models are not allowed |
| 4 | Relationship: event-to-entity must be many-to-one | When source is event and target is entity, cardinality must be `many-to-one` |
| 5 | Relationship: entity-to-event must be one-to-many | When source is entity and target is event, cardinality must be `one-to-many` |
| 6 | Relationship: entity-to-entity cardinality constraint | When both source and target are entity, cardinality must be one of: `one-to-one`, `one-to-many`, `many-to-one` |

---

## Acceptance Criteria

- [ ] All 11 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("data-graph", "rudder/v1")`
- [ ] All 6 semantic validations listed above are implemented as V1 rules
- [ ] Cardinality constraint validations cover all model type combinations (entity-entity, entity-event, event-entity, event-event)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-data-graph-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `data-graph` resource (including inline models and relationships). These rules target `rudder/v1` specs, implementing 11 syntactic and 6 semantic validations covering required fields, model type constraints, and relationship cardinality rules.

---

## Changes

* Add V1 syntactic rule for data graph spec validation (account_id, model fields, relationship fields)
* Add V1 semantic rule for reference resolution and cardinality constraint validation

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations (cardinality matrix: entity-entity, entity-event, event-entity, event-event)
* Table-driven tests covering valid and invalid V1 data graph specs

---

## Risk / Impact

Low
V1 validation is new functionality; no rules existed previously for data graph.

---

## Checklist

* [ ] Ticket linked
* [ ] Tests added/updated
* [ ] No breaking changes (or documented)
```
