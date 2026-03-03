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

Data graph uses a single spec kind (`data-graph`) with inline models and relationships. There is **no structural difference** between V0.1 and V1 -- the same structs are used. Since no V0.1 rules exist in the new validation engine for data graph, adding `validate:` tags to the shared structs is safe.

### Structs — Current (`model/datagraph.go`)

```go
type DataGraphSpec struct {
    ID        string      `json:"id" mapstructure:"id"`
    AccountID string      `json:"account_id" mapstructure:"account_id"`
    Models    []ModelSpec `json:"models,omitempty" mapstructure:"models"`
}

type ModelSpec struct {
    ID            string             `json:"id" mapstructure:"id"`
    DisplayName   string             `json:"display_name" mapstructure:"display_name"`
    Type          string             `json:"type" mapstructure:"type"`
    Table         string             `json:"table" mapstructure:"table"`
    Description   string             `json:"description,omitempty" mapstructure:"description"`
    Relationships []RelationshipSpec `json:"relationships,omitempty" mapstructure:"relationships"`
    PrimaryID     string             `json:"primary_id,omitempty" mapstructure:"primary_id"`
    Root          bool               `json:"root,omitempty" mapstructure:"root"`
    Timestamp     string             `json:"timestamp,omitempty" mapstructure:"timestamp"`
}

type RelationshipSpec struct {
    ID            string `json:"id" mapstructure:"id"`
    DisplayName   string `json:"display_name" mapstructure:"display_name"`
    Cardinality   string `json:"cardinality" mapstructure:"cardinality"`
    Target        string `json:"target" mapstructure:"target"`
    SourceJoinKey string `json:"source_join_key" mapstructure:"source_join_key"`
    TargetJoinKey string `json:"target_join_key" mapstructure:"target_join_key"`
}
```

### Structs — Updated (tags to add)

```go
type DataGraphSpec struct {
    ID        string      `json:"id" mapstructure:"id"`
    AccountID string      `json:"account_id" mapstructure:"account_id" validate:"required"`
    Models    []ModelSpec `json:"models,omitempty" mapstructure:"models" validate:"omitempty,dive"`
}

type ModelSpec struct {
    ID            string             `json:"id" mapstructure:"id" validate:"required"`
    DisplayName   string             `json:"display_name" mapstructure:"display_name" validate:"required"`
    Type          string             `json:"type" mapstructure:"type" validate:"required,oneof=entity event"`
    Table         string             `json:"table" mapstructure:"table" validate:"required"`
    Description   string             `json:"description,omitempty" mapstructure:"description"`
    Relationships []RelationshipSpec `json:"relationships,omitempty" mapstructure:"relationships" validate:"omitempty,dive"`
    PrimaryID     string             `json:"primary_id,omitempty" mapstructure:"primary_id"`
    Root          bool               `json:"root,omitempty" mapstructure:"root"`
    Timestamp     string             `json:"timestamp,omitempty" mapstructure:"timestamp"`
}

type RelationshipSpec struct {
    ID            string `json:"id" mapstructure:"id" validate:"required"`
    DisplayName   string `json:"display_name" mapstructure:"display_name" validate:"required"`
    Cardinality   string `json:"cardinality" mapstructure:"cardinality" validate:"required,oneof=one-to-one one-to-many many-to-one"`
    Target        string `json:"target" mapstructure:"target" validate:"required"`
    SourceJoinKey string `json:"source_join_key" mapstructure:"source_join_key" validate:"required"`
    TargetJoinKey string `json:"target_join_key" mapstructure:"target_join_key" validate:"required"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("data-graph", "rudder/v1")` and decode into `DataGraphSpec`.

### Tag-Based (handled by `rules.ValidateStruct()`)

#### Data Graph Level

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `account_id` required | `validate:"required"` on `AccountID` | Data graph must have a non-empty `account_id` |
| 2 | Models dive | `validate:"omitempty,dive"` on `Models` | Recursively validates each `ModelSpec` |

#### Model Level (inline within data graph)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 3 | `id` required | `validate:"required"` on `ID` | Each model must have a non-empty `id` |
| 4 | `display_name` required | `validate:"required"` on `DisplayName` | Each model must have a non-empty `display_name` |
| 5 | `type` required + oneof | `validate:"required,oneof=entity event"` on `Type` | Model type must be one of: `entity`, `event` |
| 6 | `table` required | `validate:"required"` on `Table` | Each model must have a non-empty `table` |
| 7 | Relationships dive | `validate:"omitempty,dive"` on `Relationships` | Recursively validates each `RelationshipSpec` |

#### Relationship Level (inline within model)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 8 | `id` required | `validate:"required"` on `ID` | Each relationship must have a non-empty `id` |
| 9 | `display_name` required | `validate:"required"` on `DisplayName` | Each relationship must have a non-empty `display_name` |
| 10 | `cardinality` required + oneof | `validate:"required,oneof=one-to-one one-to-many many-to-one"` on `Cardinality` | Cardinality must be a valid value |
| 11 | `target` required | `validate:"required"` on `Target` | Must reference a target model |
| 12 | `source_join_key` required | `validate:"required"` on `SourceJoinKey` | Must have a non-empty source join key |
| 13 | `target_join_key` required | `validate:"required"` on `TargetJoinKey` | Must have a non-empty target join key |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 14 | Entity model: `primary_id` required | When `type` is `entity`, `primary_id` must be non-empty |
| 15 | Event model: `timestamp` required | When `type` is `event`, `timestamp` must be non-empty |

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

- [ ] `validate:` tags added to `DataGraphSpec`, `ModelSpec`, and `RelationshipSpec` shared structs (safe since no V0.1 rules exist in new engine)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#13)
- [ ] Custom logic implemented for conditional entity/event field requirements (#14-#15)
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

* Add `validate:` tags to `DataGraphSpec`, `ModelSpec`, `RelationshipSpec` shared structs
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom logic for conditional entity/event fields
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
