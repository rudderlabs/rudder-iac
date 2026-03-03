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
| Validation tags | Has `validate:"required"`, `validate:"pattern=display_name"` | **Add matching tags** (see below) |

The structures are nearly identical. V1 structs must be updated to include the same `validate:` tags as V0.1, so that `rules.ValidateStruct()` handles tag-expressible validations automatically.

### V0.1 Struct (`Category` in `localcatalog/model.go`)

```go
type Category struct {
    LocalID string `mapstructure:"id" json:"id" validate:"required"`
    Name    string `mapstructure:"name" json:"name" validate:"required,pattern=display_name"`
}
```

### V1 Struct — Current (`CategoryV1` in `localcatalog/model.go`)

```go
type CategoryV1 struct {
    LocalID string `mapstructure:"id" json:"id"`
    Name    string `mapstructure:"name" json:"name"`
}
```

### V1 Struct — Updated (tags to add)

```go
type CategoryV1 struct {
    LocalID string `mapstructure:"id" json:"id" validate:"required"`
    Name    string `mapstructure:"name" json:"name" validate:"required,pattern=display_name"`
}
```

Also update `CategorySpecV1` to enable recursive validation via `dive`:

```go
type CategorySpecV1 struct {
    Categories []CategoryV1 `json:"categories" validate:"dive"`
}
```

The `display_name` pattern (`^[A-Z_a-z][ \w,.-]{2,64}$`) is already registered in `cli/internal/providers/datacatalog/rules/constants.go`.

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("categories", "rudder/v1")` and decode into `CategorySpecV1` / `CategoryV1`.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `LocalID` | Category must have non-empty `id` |
| 2 | `name` required | `validate:"required"` on `Name` | Category must have non-empty `name` |
| 3 | `name` matches pattern | `validate:"pattern=display_name"` on `Name` | Name must match `^[A-Z_a-z][\s\w,.-]{2,64}$` |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 4 | No leading/trailing whitespace in `name` | Category name must not have leading or trailing whitespace characters |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `name` uniqueness | No two categories may share the same name |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `CategoryV1` and `CategorySpecV1` structs matching V0.1 style
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#3)
- [ ] Custom logic implemented for whitespace check (#4)
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

* Add `validate:` tags to `CategoryV1` and `CategorySpecV1` structs
* Add V1 syntactic rule for category spec validation using `rules.ValidateStruct()` for tag-based validations + custom whitespace check
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
