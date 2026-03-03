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

The RETL SQL model spec structure is **identical** between V0.1 and V1. The same `SQLModelSpec` struct is used for both versions.

| Aspect | V0.1 | V1 |
|--------|------|----|
| Struct | `SQLModelSpec` | `SQLModelSpec` (same) |
| Fields | Identical | Identical |

### Struct (`SQLModelSpec` in `sqlmodel/model.go`)

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

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("retl-source-sql-model", "rudder/v1")` and decode into `SQLModelSpec`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id` required | SQL model must have a non-empty `id` field |
| 2 | `display_name` required | SQL model must have a non-empty `display_name` field |
| 3 | `account_id` required | SQL model must have a non-empty `account_id` field |
| 4 | `primary_key` required | SQL model must have a non-empty `primary_key` field |
| 5 | `source_definition` required and in allowed set | Must be one of: postgres, redshift, snowflake, bigquery, mysql, databricks, trino |
| 6 | `sql` and `file` mutually exclusive, one required | Either `sql` or `file` must be specified, but not both |
| 7 | `sql` non-empty after file resolution | After resolving `file` to `sql` content, the resulting SQL must not be empty |

---

## Semantic Validations to Add for V1

No semantic validations needed. RETL SQL models have no cross-resource references.

---

## Acceptance Criteria

- [ ] All 7 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("retl-source-sql-model", "rudder/v1")`
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

* Add V1 syntactic rule for SQL model spec validation (required fields, source_definition in allowed set, sql/file exclusivity, sql non-empty)

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
