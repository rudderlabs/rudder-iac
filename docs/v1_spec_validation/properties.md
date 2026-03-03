# Properties - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **exist** in the new validation engine. Three rules are registered via `LegacyVersionPatterns("properties")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `datacatalog/properties/spec-syntax-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/property/property_spec_valid.go` |
| `datacatalog/properties/config-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/property/property_config_valid.go` |
| `datacatalog/properties/semantic-valid` | Semantic | `cli/internal/providers/datacatalog/rules/property/property_semantic_valid.go` |

These rules decode into V0 structs (`PropertySpec` / `Property`) and do **not** match V1 specs (`rudder/v1`).

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`Property`) | V1 (`PropertyV1`) |
|--------|--------------------|--------------------|
| Config field name | `propConfig` (camelCase JSON key) | `config` (snake_case JSON key) |
| Config keys | camelCase (`itemTypes`, `minLength`, `maxLength`, `exclusiveMinimum`) | snake_case (`item_types`, `min_length`, `max_length`, `exclusive_minimum`) |
| Multiple types | Comma-separated in `type` field (e.g. `"string,number"`) | Separate `types` array field (e.g. `["string", "number"]`) |
| Item types | Inside config as `itemTypes` | Top-level `item_type` (single) and `item_types` (array) fields |
| Validation tags on struct | Has `validate:"required"`, `validate:"gte=1,lte=65"` etc. | No validation tags on V1 struct |
| Ref format for custom types | `#/custom-types/<group>/<id>` | `#custom-type:<id>` |

### V0.1 Struct (`Property` in `localcatalog/model.go`)

```go
type Property struct {
    LocalID     string                 `mapstructure:"id" json:"id" validate:"required"`
    Name        string                 `mapstructure:"name" json:"name" validate:"required,gte=1,lte=65"`
    Description string                 `mapstructure:"description,omitempty" json:"description" validate:"omitempty,gte=3,lte=2000"`
    Type        string                 `mapstructure:"type,omitempty" json:"type"`
    Config      map[string]interface{} `mapstructure:"propConfig,omitempty" json:"propConfig,omitempty"`
}
```

### V1 Struct (`PropertyV1` in `localcatalog/model_v1.go`)

```go
type PropertyV1 struct {
    LocalID     string                 `mapstructure:"id" json:"id"`
    Name        string                 `mapstructure:"name" json:"name"`
    Description string                 `mapstructure:"description,omitempty" json:"description,omitempty"`
    Type        string                 `mapstructure:"type,omitempty" json:"type,omitempty"`
    Types       []string               `mapstructure:"types,omitempty" json:"types,omitempty"`
    ItemType    string                 `mapstructure:"item_type,omitempty" json:"item_type,omitempty"`
    ItemTypes   []string               `mapstructure:"item_types,omitempty" json:"item_types,omitempty"`
    Config      map[string]interface{} `mapstructure:"config,omitempty" json:"config,omitempty"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("properties", "rudder/v1")` and decode into `PropertySpecV1` / `PropertyV1`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id` required | Property must have a non-empty `id` field |
| 2 | `name` required | Property must have a non-empty `name` field |
| 3 | `type` and `types` mutually exclusive | Only one of `type` (single) or `types` (array) can be specified |
| 4 | `item_type` and `item_types` mutually exclusive | Only one of `item_type` (single) or `item_types` (array) can be specified |
| 5 | No leading/trailing whitespace in `name` | Property name must not have leading or trailing whitespace |
| 6 | `types` array values in ValidTypes | Each value in `types` must be one of: string, number, integer, boolean, null, array, object |
| 7 | No duplicate values in `types` array | The `types` array must not contain duplicate type values |
| 8 | Custom-type in `type` disallows `config` | When `type` references a custom type (`#custom-type:<id>`), `config` must be nil |
| 9 | No comma-separated values in `type` | The `type` field must not contain comma-separated values; use `types` array instead |
| 10 | `type` must be in ValidTypes or custom-type ref | Single `type` must be a valid primitive type or a custom-type reference (`#custom-type:<id>`) |
| 11 | `item_type` must be in ValidTypes or custom-type ref | `item_type` must be a valid primitive type or custom-type reference |
| 12 | `item_types` with custom-type must be single element | If `item_types` array contains a custom-type ref, it must be the only element |
| 13 | Config validation per type | Validate `config` fields based on the property type (snake_case keys: `min_length`, `max_length`, `pattern`, `format` for string; `minimum`, `maximum`, `exclusive_minimum`, `exclusive_maximum`, `multiple_of` for number/integer; `item_types`, `min_items`, `max_items`, `unique_items` for array; etc.) |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Custom-type refs in `type` resolve | If `type` is `#custom-type:<id>`, the referenced custom type must exist in the resource graph |
| 2 | Custom-type refs in `item_type` resolve | If `item_type` is a custom-type ref, it must exist in the graph |
| 3 | Custom-type refs in `item_types` resolve | Each custom-type ref in `item_types` must exist in the graph |
| 4 | No custom-type refs in `types` array | The `types` array must not contain custom-type references (use single `type` field instead) |
| 5 | Config `item_types` custom-type refs resolve | Custom-type references within config `item_types` must resolve in the graph |
| 6 | `(name, type, itemTypes)` uniqueness | No two properties may share the same combination of (name, type, itemTypes) |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] All 13 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("properties", "rudder/v1")` and decoding into `PropertySpecV1`
- [ ] All 6 semantic validations listed above are implemented as V1 rules
- [ ] Config validation handles snake_case keys (`min_length`, `max_length`, `item_types`, etc.) for V1 specs
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-properties-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `properties` resource in the datacatalog provider. These rules target `rudder/v1` specs and decode into `PropertySpecV1`/`PropertyV1` structs, implementing 13 syntactic and 6 semantic validations.

---

## Changes

* Add V1 syntactic rule for property spec validation (id, name, type/types exclusivity, config validation)
* Add V1 semantic rule for property reference resolution and uniqueness
* Add V1 config validation handling snake_case keys

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 property specs

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
