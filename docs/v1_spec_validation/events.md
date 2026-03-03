# Events - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **exist** in the new validation engine. Two rules are registered via `LegacyVersionPatterns("events")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `datacatalog/events/spec-syntax-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/event/event_spec_valid.go` |
| `datacatalog/events/semantic-valid` | Semantic | `cli/internal/providers/datacatalog/rules/event/event_semantic_valid.go` |

These rules decode into V0 structs (`EventSpec` / `Event`) and do **not** match V1 specs (`rudder/v1`).

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`Event`) | V1 (`EventV1`) |
|--------|-----------------|-----------------|
| Validation tags | Has `validate:"required"`, `validate:"oneof=track screen identify group page"` etc. | No validation tags |
| Category ref format | `#/categories/<group>/<id>` (validated by `pattern=legacy_category_ref`) | `#category:<id>` |
| Description validation | `validate:"omitempty,gte=3,lte=2000,pattern=letter_start"` | No tags |

The structural shape is very similar between V0.1 and V1 -- the main difference is the reference format for categories and the absence of validation tags on V1 structs.

### V0.1 Struct (`Event` in `localcatalog/model.go`)

```go
type Event struct {
    LocalID     string  `json:"id" mapstructure:"id" validate:"required"`
    Name        string  `json:"name" mapstructure:"name,omitempty"`
    Type        string  `json:"event_type" mapstructure:"event_type" validate:"oneof=track screen identify group page"`
    Description string  `json:"description" mapstructure:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    CategoryRef *string `json:"category" mapstructure:"category,omitempty" validate:"omitempty,pattern=legacy_category_ref"`
}
```

### V1 Struct (`EventV1` in `localcatalog/model.go`)

```go
type EventV1 struct {
    LocalID     string  `json:"id" mapstructure:"id"`
    Name        string  `json:"name,omitempty" mapstructure:"name,omitempty"`
    Type        string  `json:"event_type" mapstructure:"event_type"`
    Description string  `json:"description,omitempty" mapstructure:"description,omitempty"`
    CategoryRef *string `json:"category,omitempty" mapstructure:"category,omitempty"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("events", "rudder/v1")` and decode into `EventSpecV1` / `EventV1`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id` required | Event must have a non-empty `id` field |
| 2 | `event_type` required | Event must have a non-empty `event_type` field, value in `track`, `screen`, `identify`, `group`, `page` |
| 3 | Track events: `name` required, 1-64 chars | When `event_type` is `track`, `name` is required and must be between 1 and 64 characters |
| 4 | Non-track events: `name` must be empty | When `event_type` is not `track`, `name` must be empty |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Category ref resolves | If `category` is set (format `#category:<id>`), the referenced category must exist in the resource graph |
| 2 | `(name, eventType)` uniqueness | No two events may share the same combination of (name, event_type) |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] All 4 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("events", "rudder/v1")` and decoding into `EventSpecV1`
- [ ] All 2 semantic validations listed above are implemented as V1 rules
- [ ] Category ref validation uses V1 ref format (`#category:<id>`)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-events-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `events` resource in the datacatalog provider. These rules target `rudder/v1` specs and decode into `EventSpecV1`/`EventV1` structs, implementing 4 syntactic and 2 semantic validations.

---

## Changes

* Add V1 syntactic rule for event spec validation (id, event_type, name constraints per event type)
* Add V1 semantic rule for category reference resolution and (name, eventType) uniqueness

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 event specs

---

## Risk / Impact

Low
V1 validation is new functionality; does not affect existing V0.1 validation paths.

---

## Checklist

* [ ] Ticket linked
* [ ] Tests added/updated
* [ ] No breaking changes (or documented)
```
