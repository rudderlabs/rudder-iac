# Transformation Library - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

**No validation rules exist** in the new validation engine for transformation library resources (any version). The transformations provider does not have a `rules/` directory and does not implement `RuleHandler`.

All existing validations live in the old handler-based framework:
- `cli/internal/providers/transformations/handlers/library/handler.go` (ValidateSpec, ValidateResource)

---

## Structural Differences: V0.1 vs V1

There is **no structural difference** between V0.1 and V1. The same `TransformationLibrarySpec` struct is used for both versions.

### Struct (`TransformationLibrarySpec` in `project/specs/transformation.go`)

```go
type TransformationLibrarySpec struct {
    ID          string `json:"id" mapstructure:"id"`
    Name        string `json:"name" mapstructure:"name"`
    Description string `json:"description" mapstructure:"description"`
    Language    string `json:"language" mapstructure:"language"`
    Code        string `json:"code,omitempty" mapstructure:"code"`
    File        string `json:"file,omitempty" mapstructure:"file"`
    ImportName  string `json:"import_name" mapstructure:"import_name"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("transformation-library", "rudder/v1")` and decode into `TransformationLibrarySpec`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id` required | Library must have a non-empty `id` field |
| 2 | `name` required | Library must have a non-empty `name` field |
| 3 | `import_name` required | Library must have a non-empty `import_name` field |
| 4 | `code` and `file` mutually exclusive, one required | Either `code` or `file` must be specified, but not both |
| 5 | `language` required | Library must have a non-empty `language` field |
| 6 | `language` in allowed values | Language must be one of: `javascript`, `python` |
| 7 | `import_name` must equal camelCase of `name` | The `import_name` field must be the camelCase transformation of the `name` field |
| 8 | Code syntax validation | When `code` is provided (or resolved from `file`), validate syntax using the appropriate parser (esbuild for JavaScript, Python parser for Python) |

---

## Semantic Validations to Add for V1

No semantic validations needed. Transformation libraries have no cross-resource references.

---

## Acceptance Criteria

- [ ] All 8 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("transformation-library", "rudder/v1")`
- [ ] `import_name` camelCase validation matches the existing handler logic
- [ ] Code syntax validation integrates with the existing parser infrastructure (`parser.ValidateSyntax()`)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-transformation-library-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `transformation-library` resource. These rules target `rudder/v1` specs, implementing 8 syntactic validations including required fields, language constraints, code/file exclusivity, import_name camelCase validation, and code syntax validation.

---

## Changes

* Add V1 syntactic rule for library spec validation (required fields, language, code/file exclusivity)
* Add V1 syntactic rule for import_name camelCase validation and code syntax validation

---

## Testing

* Unit tests for all syntactic validations
* Table-driven tests covering valid and invalid V1 library specs
* Tests for import_name camelCase validation
* Tests for code syntax validation (valid/invalid JavaScript and Python)

---

## Risk / Impact

Low
V1 validation is new functionality; no rules existed previously for transformation libraries.

---

## Checklist

* [ ] Ticket linked
* [ ] Tests added/updated
* [ ] No breaking changes (or documented)
```
