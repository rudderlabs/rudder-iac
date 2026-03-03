# RETL SQL Model - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **partially exist** in the new validation engine. One rule is registered via `LegacyVersionPatterns("retl-source-sql-model")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `retl/sqlmodel/spec-syntax-valid` | Syntactic | `cli/internal/providers/retl/rules/sqlmodel/sqlmodel_spec_valid.go` |

Additionally, the handler (`sqlmodel/handler.go`) performs checks during `LoadSpec` and `ValidateSQLModelResource` that are **not ported** to the new engine for any version:
- Unknown fields rejection (`ErrorUnused: true`)
- `sql` non-empty after file resolution

No semantic rules exist for RETL in either framework. This rule does **not** match V1 specs (`rudder/v1`).

---

## Structural Differences: V0.1 vs V1

The RETL SQL model spec structure is **identical** between V0.1 and V1. The same `SQLModelSpec` struct is used for both versions. The struct **already has `validate:` tags** covering most validations.

| Aspect | V0.1 | V1 |
|--------|------|----|
| Struct | `SQLModelSpec` | `SQLModelSpec` (same) |
| Fields | Identical | Identical |
| Validation tags | **Already present** | **Already present** (shared struct) |

### Struct (`SQLModelSpec` in `sqlmodel/model.go`) — Already Has Tags

```go
type SQLModelSpec struct {
    ID               string           `json:"id"                mapstructure:"id"                validate:"required"`
    DisplayName      string           `json:"display_name"      mapstructure:"display_name"      validate:"required"`
    Description      string           `json:"description"       mapstructure:"description"`
    File             *string          `json:"file"              mapstructure:"file"`
    SQL              *string          `json:"sql"               mapstructure:"sql"               validate:"required_without=File,excluded_with=File"`
    AccountID        string           `json:"account_id"        mapstructure:"account_id"        validate:"required"`
    PrimaryKey       string           `json:"primary_key"       mapstructure:"primary_key"       validate:"required"`
    SourceDefinition SourceDefinition `json:"source_definition" mapstructure:"source_definition" validate:"required,oneof=postgres redshift snowflake bigquery mysql databricks trino"`
    Enabled          *bool            `json:"enabled"           mapstructure:"enabled"`
}
```

**No struct tag changes needed.** The existing tags already cover validations #1-#6. The V1 rule should use `rules.ValidateStruct()` to leverage these existing tags.

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("retl-source-sql-model", "rudder/v1")` and decode into `SQLModelSpec`.

### Tag-Based (handled by `rules.ValidateStruct()` via existing tags)

| # | Validation | Existing Tag | Description |
|---|-----------|--------------|-------------|
| 1 | `id` required | `validate:"required"` on `ID` | SQL model must have a non-empty `id` |
| 2 | `display_name` required | `validate:"required"` on `DisplayName` | SQL model must have a non-empty `display_name` |
| 3 | `account_id` required | `validate:"required"` on `AccountID` | SQL model must have a non-empty `account_id` |
| 4 | `primary_key` required | `validate:"required"` on `PrimaryKey` | SQL model must have a non-empty `primary_key` |
| 5 | `source_definition` required + oneof | `validate:"required,oneof=postgres redshift snowflake bigquery mysql databricks trino"` on `SourceDefinition` | Must be one of the allowed values |
| 6 | `sql`/`file` mutual exclusivity | `validate:"required_without=File,excluded_with=File"` on `SQL` | Either `sql` or `file` must be specified, but not both |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 7 | `sql` non-empty after file resolution | After resolving `file` to `sql` content, the resulting SQL must not be empty |

---

## Semantic Validations to Add for V1

No semantic validations needed. RETL SQL models have no cross-resource references.

---

## Acceptance Criteria

- [ ] No struct tag changes needed (existing tags on `SQLModelSpec` already cover #1-#6)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` to leverage existing tags for validations #1-#6
- [ ] Custom logic implemented for sql-non-empty-after-file-resolution (#7)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-retl-sql-model-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `retl-source-sql-model` resource. These rules target `rudder/v1` specs, implementing 7 syntactic validations including required fields, source definition validation, and sql/file mutual exclusivity.

---

## Changes

* Add V1 syntactic rule using `rules.ValidateStruct()` to leverage existing struct tags for validations #1-#6 + custom logic for sql non-empty after file resolution

---

## Testing

* Unit tests for all syntactic validations
* Table-driven tests covering valid and invalid V1 SQL model specs

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
