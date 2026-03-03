# Categories - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **exist** in the new validation engine. Two rules are registered via `LegacyVersionPatterns("categories")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `datacatalog/categories/spec-syntax-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/category/category_spec_valid.go` |
| `datacatalog/categories/semantic-valid` | Semantic | `cli/internal/providers/datacatalog/rules/category/category_semantic_valid.go` |

These rules decode into V0 structs (`CategorySpec` / `Category`) and do **not** match V1 specs (`rudder/v1`).

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`Category`) | V1 (`CategoryV1`) |
|--------|---------------------|--------------------|
| Validation tags | Has `validate:"required"`, `validate:"pattern=display_name"` | No validation tags |

The structures are nearly identical. The only difference is the presence of go-validator tags on V0.1 structs.

### V0.1 Struct (`Category` in `localcatalog/model.go`)

```go
type Category struct {
    LocalID string `mapstructure:"id" json:"id" validate:"required"`
    Name    string `mapstructure:"name" json:"name" validate:"required,pattern=display_name"`
}
```

### V1 Struct (`CategoryV1` in `localcatalog/model.go`)

```go
type CategoryV1 struct {
    LocalID string `mapstructure:"id" json:"id"`
    Name    string `mapstructure:"name" json:"name"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("categories", "rudder/v1")` and decode into `CategorySpecV1` / `CategoryV1`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id` and `name` required | Category must have non-empty `id` and `name` fields |
| 2 | No leading/trailing whitespace in `name` | Category name must not have leading or trailing whitespace characters |
| 3 | `name` matches pattern | Name must match `^[A-Z_a-z][\s\w,.-]{2,64}$` (starts with letter or underscore, 2-64 chars, allows spaces/word chars/commas/periods/hyphens) |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `name` uniqueness | No two categories may share the same name |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] All 3 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("categories", "rudder/v1")` and decoding into `CategorySpecV1`
- [ ] The 1 semantic validation listed above is implemented as a V1 rule
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-categories-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `categories` resource in the datacatalog provider. These rules target `rudder/v1` specs and decode into `CategorySpecV1`/`CategoryV1` structs, implementing 3 syntactic and 1 semantic validation.

---

## Changes

* Add V1 syntactic rule for category spec validation (required fields, name whitespace, name pattern)
* Add V1 semantic rule for category name uniqueness

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 category specs

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
