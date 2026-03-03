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
| Validation tags | Has `validate:"required"`, `validate:"oneof=track screen identify group page"` etc. | **Add matching tags** (see below) |
| Category ref format | `#/categories/<group>/<id>` (validated by `pattern=legacy_category_ref`) | `#category:<id>` (use `pattern=category_ref`) |
| Description validation | `validate:"omitempty,gte=3,lte=2000,pattern=letter_start"` | **Add matching tag** |

The structural shape is very similar between V0.1 and V1. V1 structs must be updated to include `validate:` tags so that `rules.ValidateStruct()` handles tag-expressible validations automatically. The main difference is the V1 reference format for categories (`category_ref` instead of `legacy_category_ref`).

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

### V1 Struct — Current (`EventV1` in `localcatalog/model.go`)

```go
type EventV1 struct {
    LocalID     string  `json:"id" mapstructure:"id"`
    Name        string  `json:"name,omitempty" mapstructure:"name,omitempty"`
    Type        string  `json:"event_type" mapstructure:"event_type"`
    Description string  `json:"description,omitempty" mapstructure:"description,omitempty"`
    CategoryRef *string `json:"category,omitempty" mapstructure:"category,omitempty"`
}
```

### V1 Struct — Updated (tags to add)

```go
type EventV1 struct {
    LocalID     string  `json:"id" mapstructure:"id" validate:"required"`
    Name        string  `json:"name,omitempty" mapstructure:"name,omitempty"`
    Type        string  `json:"event_type" mapstructure:"event_type" validate:"required,oneof=track screen identify group page"`
    Description string  `json:"description,omitempty" mapstructure:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    CategoryRef *string `json:"category,omitempty" mapstructure:"category,omitempty" validate:"omitempty,pattern=category_ref"`
}
```

Also update `EventSpecV1` to enable recursive validation via `dive`:

```go
type EventSpecV1 struct {
    Events []EventV1 `json:"events" validate:"dive"`
}
```

The `category_ref` pattern (`#categories:<id>`) and `letter_start` pattern are already registered in `cli/internal/providers/datacatalog/rules/constants.go` and `cli/internal/provider/rules/funcs/init.go` respectively.

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("events", "rudder/v1")` and decode into `EventSpecV1` / `EventV1`.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `LocalID` | Event must have a non-empty `id` |
| 2 | `event_type` required + oneof | `validate:"required,oneof=track screen identify group page"` on `Type` | Event type must be one of the allowed values |
| 3 | `description` constraints | `validate:"omitempty,gte=3,lte=2000,pattern=letter_start"` on `Description` | If present, 3-2000 chars starting with a letter |
| 4 | `category` ref format | `validate:"omitempty,pattern=category_ref"` on `CategoryRef` | If present, must match `#categories:<id>` |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 5 | Track events: `name` required, 1-64 chars | When `event_type` is `track`, `name` is required and must be between 1 and 64 characters |
| 6 | Non-track events: `name` must be empty | When `event_type` is not `track`, `name` must be empty |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Category ref resolves | If `category` is set (format `#category:<id>`), the referenced category must exist in the resource graph |
| 2 | `(name, eventType)` uniqueness | No two events may share the same combination of (name, event_type) |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `EventV1` and `EventSpecV1` structs matching V0.1 style (with V1 ref patterns)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#4)
- [ ] Custom logic implemented for conditional name validations (#5-#6)
- [ ] All 2 semantic validations listed above are implemented as V1 rules
- [ ] Category ref validation uses V1 ref pattern (`category_ref` = `#categories:<id>`)
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

* Add `validate:` tags to `EventV1` and `EventSpecV1` structs (using V1 ref patterns)
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom name-per-event-type logic
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
