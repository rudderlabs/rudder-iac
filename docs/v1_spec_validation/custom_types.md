# Custom Types - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **exist** in the new validation engine. Three rules are registered via `LegacyVersionPatterns("custom-types")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `datacatalog/custom-types/spec-syntax-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/customtype/customtype_spec_valid.go` |
| `datacatalog/custom-types/config-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/customtype/customtype_config_valid.go` |
| `datacatalog/custom-types/semantic-valid` | Semantic | `cli/internal/providers/datacatalog/rules/customtype/customtype_semantic_valid.go` |

These rules decode into V0 structs (`CustomTypeSpec` / `CustomType`) and do **not** match V1 specs (`rudder/v1`).

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`CustomType`) | V1 (`CustomTypeV1`) |
|--------|----------------------|----------------------|
| Property reference field | `$ref` (`json:"$ref"`) | `property` (`json:"property"`) |
| Property ref format | `#/properties/<group>/<id>` (validated by `pattern=legacy_property_ref`) | `#property:<id>` |
| Variant property refs | `PropertyReference` with `$ref` field | `PropertyReferenceV1` with `property` field |
| Config keys | camelCase (`itemTypes`, `minLength`) | snake_case (`item_types`, `min_length`) |
| Validation tags | Has `validate:"required"`, `validate:"pattern=custom_type_name"` etc. | No validation tags on V1 struct |

### V0.1 Struct (`CustomType` in `localcatalog/model.go`)

```go
type CustomType struct {
    LocalID     string               `mapstructure:"id" json:"id" validate:"required"`
    Name        string               `mapstructure:"name" json:"name" validate:"required,gte=2,lte=65,pattern=custom_type_name"`
    Description string               `mapstructure:"description,omitempty" json:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    Type        string               `mapstructure:"type" json:"type" validate:"required,pattern=primitive_type"`
    Config      map[string]any       `mapstructure:"config,omitempty" json:"config,omitempty"`
    Properties  []CustomTypeProperty `mapstructure:"properties,omitempty" json:"properties,omitempty" validate:"omitempty,dive"`
    Variants    Variants             `mapstructure:"variants,omitempty" json:"variants,omitempty" validate:"excluded_unless=Type object,omitempty,max=1,dive"`
}

type CustomTypeProperty struct {
    Ref      string `mapstructure:"$ref" json:"$ref" validate:"required,pattern=legacy_property_ref"`
    Required bool   `mapstructure:"required" json:"required"`
}
```

### V1 Struct (`CustomTypeV1` in `localcatalog/model_v1.go`)

```go
type CustomTypeV1 struct {
    LocalID     string                 `mapstructure:"id" json:"id"`
    Name        string                 `mapstructure:"name" json:"name"`
    Description string                 `mapstructure:"description,omitempty" json:"description,omitempty"`
    Type        string                 `mapstructure:"type" json:"type"`
    Config      map[string]any         `mapstructure:"config,omitempty" json:"config,omitempty"`
    Properties  []CustomTypePropertyV1 `mapstructure:"properties,omitempty" json:"properties,omitempty"`
    Variants    VariantsV1             `mapstructure:"variants,omitempty" json:"variants,omitempty"`
}

type CustomTypePropertyV1 struct {
    Property string `mapstructure:"property" json:"property"`
    Required bool   `mapstructure:"required" json:"required"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("custom-types", "rudder/v1")` and decode into `CustomTypeSpecV1` / `CustomTypeV1`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id`, `name`, `type` required | Custom type must have non-empty `id`, `name`, and `type` fields |
| 2 | `name` matches pattern | Name must match `^[A-Z][a-zA-Z0-9_-]{2,64}$` (starts with capital letter, 2-65 chars total, letters/numbers/underscores/dashes) |
| 3 | `type` in ValidTypes | Type must be one of: string, number, integer, boolean, null, array, object |
| 4 | `type == "object"`: no `config` | When type is object, `config` must be nil/empty |
| 5 | `type == "object"`: properties must have `property` field | Each property in `properties` array must have non-empty `property` field |
| 6 | Config validation per type | Config fields validated based on type (snake_case keys: `min_length`/`max_length`/`format`/`pattern` for string; `minimum`/`maximum`/`exclusive_minimum`/`exclusive_maximum`/`multiple_of` for number/integer; `item_types`/`min_items`/`max_items`/`unique_items` for array) |
| 7 | `variants` only when `type == "object"` | Variants array is only allowed when type is object |
| 8 | Variant structure validation | Max 1 variant; variant `type` must be `"discriminator"`; `discriminator` field required; `cases` must have at least 1 element; each case needs `display_name`, `match` (min 1 element, values must be string/bool/integer), `properties` (min 1 element with `property` field); default properties must have `property` field |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Property refs resolve | Each `property` field in `properties` array (format `#property:<id>`) must resolve to an existing property in the graph |
| 2 | Variant discriminator ref resolves with type constraint | Discriminator (format `#property:<id>`) must resolve to a property with type string, integer, or boolean |
| 3 | Variant case/default property refs resolve | All property references in variant cases and default must resolve in the graph |
| 4 | Config `item_types` custom-type refs resolve | Custom-type references in config `item_types` (snake_case) must exist in the graph |
| 5 | `name` uniqueness | No two custom types may share the same name |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] All 8 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("custom-types", "rudder/v1")` and decoding into `CustomTypeSpecV1`
- [ ] All 5 semantic validations listed above are implemented as V1 rules
- [ ] Property references use V1 format (`#property:<id>` via `property` field, not `$ref`)
- [ ] Config validation handles snake_case keys for V1 specs
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-custom-types-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `custom-types` resource in the datacatalog provider. These rules target `rudder/v1` specs and decode into `CustomTypeSpecV1`/`CustomTypeV1` structs, implementing 8 syntactic and 5 semantic validations.

---

## Changes

* Add V1 syntactic rule for custom type spec validation (required fields, name pattern, type constraints, variant structure)
* Add V1 semantic rule for property reference resolution, variant discriminator validation, and name uniqueness
* Add V1 config validation handling snake_case keys

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 custom type specs

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
