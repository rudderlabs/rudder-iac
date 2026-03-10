# Transformation Library - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

**No validation rules exist** in the new validation engine for transformation library resources (any version). The transformations provider does not have a `rules/` directory and does not implement `RuleHandler`.

All existing validations live in the old handler-based framework:
- `cli/internal/providers/transformations/handlers/library/handler.go` (ValidateSpec, ValidateResource)

---

## Structural Differences: V0.1 vs V1

There is **no structural difference** between V0.1 and V1. The same `TransformationLibrarySpec` struct is used for both versions. Since no V0.1 rules exist in the new validation engine for transformation libraries, adding `validate:` tags to the shared struct is safe.

### Struct — Current (`TransformationLibrarySpec` in `project/specs/transformation.go`)

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

### Struct — Updated (tags to add)

```go
type TransformationLibrarySpec struct {
    ID          string `json:"id" mapstructure:"id" validate:"required"`
    Name        string `json:"name" mapstructure:"name" validate:"required"`
    Description string `json:"description" mapstructure:"description"`
    Language    string `json:"language" mapstructure:"language" validate:"required,oneof=javascript python"`
    Code        string `json:"code,omitempty" mapstructure:"code" validate:"required_without=File,excluded_with=File"`
    File        string `json:"file,omitempty" mapstructure:"file" validate:"required_without=Code,excluded_with=Code"`
    ImportName  string `json:"import_name" mapstructure:"import_name" validate:"required"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("transformation-library", "rudder/v1")` and decode into `TransformationLibrarySpec`.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `ID` | Library must have a non-empty `id` |
| 2 | `name` required | `validate:"required"` on `Name` | Library must have a non-empty `name` |
| 3 | `import_name` required | `validate:"required"` on `ImportName` | Library must have a non-empty `import_name` |
| 4 | `language` required + oneof | `validate:"required,oneof=javascript python"` on `Language` | Language must be one of the allowed values |
| 5 | `code`/`file` mutual exclusivity | `validate:"required_without=File,excluded_with=File"` on `Code`; `validate:"required_without=Code,excluded_with=Code"` on `File` | Exactly one of `code` or `file` must be specified — both missing and both present are invalid. Pattern mirrors `cli/internal/project/specs/metadata.go` (`LocalID`/`URN`) and `cli/internal/providers/retl/sqlmodel/model.go` (`SQL`/`File`) |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 6 | `import_name` must equal camelCase of `name` | The `import_name` field must be the camelCase transformation of the `name` field |
| 7 | Code syntax validation | When `code` is provided (or resolved from `file`), validate syntax using the appropriate parser (esbuild for JavaScript, Python parser for Python) |

---

## Semantic Validations to Add for V1

No semantic validations needed. Transformation libraries have no cross-resource references.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `TransformationLibrarySpec` shared struct (safe since no V0.1 rules exist in new engine)
- [ ] `code`/`file` mutual exclusivity expressed via `required_without`/`excluded_with` tags on both `Code` and `File` fields (no custom logic needed)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#5)
- [ ] Custom logic implemented for import_name camelCase check (#6) and code syntax validation (#7)
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

Add V1 spec validation rules for the `transformation-library` resource. These rules target `rudder/v1` specs, implementing 7 syntactic validations including required fields, language constraints, code/file exclusivity via go validator tags, import_name camelCase validation, and code syntax validation.

---

## Changes

* Add `validate:` tags to `TransformationLibrarySpec` shared struct, including `required_without`/`excluded_with` tags for `code`/`file` mutual exclusivity
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom logic for import_name camelCase and code syntax validation

---

## Testing

* Unit tests for all syntactic validations
* Table-driven tests covering valid and invalid V1 library specs
* Tests for code/file mutual exclusivity via tag validation
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
