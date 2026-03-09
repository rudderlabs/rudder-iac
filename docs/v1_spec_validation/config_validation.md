# V1 Config Validation for Properties and Custom Types

**Status:** Draft  
**Owner:** Data Catalog Validation  
**Date:** 2026-03-08

---

## 1. Overview

### Problem Statement

The shared config validator in `cli/internal/providers/datacatalog/rules/config/` currently validates the V0 config shape only:

- type-specific keys are hard-coded in camelCase (`minLength`, `itemTypes`, `exclusiveMinimum`)
- custom type references are recognized using the legacy reference format only
- the validator has no concept of spec version or config dialect

V1 config validation must support the following changes without duplicating the entire validator stack:

1. V1 config keys use snake_case instead of camelCase
2. V1 array custom type references should recognize the current custom type reference format: `#custom-types:<id>`
3. V1 key handling should be driven by caller-supplied aliasing rather than a dedicated version-specific validator fork

This spec defines a focused change in the shared `rules/config` package only. Wiring V1 property and custom type rules to consume the new behavior is intentionally deferred to follow-up PRs.

### Goals

1. Reuse the existing validator architecture and union/cross-field semantics
2. Add version-aware field normalization and custom-type-ref recognition
3. Keep V0 behavior unchanged
4. Allow V1 callers to supply key aliases for snake_case config input
5. Keep the implementation narrow enough that later rule wiring is straightforward

### Non-Goals

1. Wiring V1 property rules to the new config validator
2. Wiring V1 custom type rules to the new config validator
3. V1 semantic validation for properties or custom types
4. Validation of top-level V1 property fields such as `item_type` and `item_types`
5. Refactoring all validators into a new abstraction from scratch

---

## 2. Current State

### Existing Shared Validator

The current shared engine in `cli/internal/providers/datacatalog/rules/config/validator.go` already provides the correct high-level behavior:

- field-level union semantics across multiple types
- cross-field validation with deduplication
- validator overrides for context-specific object handling

This is the design that should be preserved.

### Current Limitations

The current implementation is V0-oriented in a few specific places:

1. Allowed-key maps are camelCase-oriented:
   - `StringTypeConfig`: `minLength`, `maxLength`
   - `IntegerTypeConfig`: `exclusiveMinimum`, `multipleOf`
   - `ArrayTypeConfig`: `itemTypes`, `minItems`, `uniqueItems`
2. Cross-field lookups are also camelCase-oriented
3. Custom type references in the shared package use the legacy matcher only
4. The validator cannot distinguish between V0 and V1 config input
5. The custom type object override currently accepts `additionalProperties`, which is not the correct V1 spelling

---

## 3. Design Decisions

### D-1: Introduce `ConfigKeyword` as the Shared Logical Keyword Layer

The shared validator flow should use `ConfigKeyword` values as the logical representation of supported config fields.

Examples:

- `KeywordMinLength`
- `KeywordItemTypes`
- `KeywordExclusiveMinimum`
- `KeywordAdditionalProperties`

This avoids leaking raw V0 camelCase field names into the new V1 aliasing API and gives the validator a version-independent keyword vocabulary.

### D-2: Add an Options-Driven Field Mapping Layer at the Boundary

Version support should be implemented by normalizing accepted raw input keys to shared `ConfigKeyword` values before invoking type validators.

Examples:

- V0 `minLength` -> `KeywordMinLength`
- V0 `itemTypes` -> `KeywordItemTypes`
- V1 `min_length` -> `KeywordMinLength`
- V1 `item_types` -> `KeywordItemTypes`
- V1 `multiple_of` -> `KeywordMultipleOf`
- V1 `additional_properties` -> `KeywordAdditionalProperties`

This field mapping should be configured through a `WithOptions` framework rather than a separate dialect interface.

### D-3: No Special Wrong-Spelling Handling in Shared Validation

This change should not introduce a special wrong-spelling path for V1 keys.

Examples:

- `min_length` can be aliased to the shared validator keyword for min length
- `minLength` is not handled specially by the shared package
- if `minLength` is not aliased, it should fall through to the existing unknown-key / not-applicable behavior

### D-4: V1 Custom Type Refs Use `#custom-types:<id>`

For this change, the current reference format is:

```text
#custom-types:<id>
```

This matches the existing `CustomTypeReferenceRegex` in `constants.go`, which uses `KindCustomTypes` (`"custom-types"`) with the pattern `^#(%s):([a-zA-Z0-9_-]+)$`.

Legacy refs such as `#/custom-types/<group>/<id>` do not need a special V1 error. If they appear inside V1 config, it is acceptable for them to fail through the existing invalid-type path.

### D-5: Scope is Limited to `config`

This work covers only config-object validation in the shared package.

It does **not** validate V1 top-level property fields such as:

- `item_type`
- `item_types`
- `type`
- `types`

Those concerns belong to property/custom-type V1 rule wiring and semantic validation, which are explicitly out of scope for this spec.

### D-6: Preserve Backward Compatibility for Existing Callers

Because V1 rule wiring is deferred, the shared package should expose the new behavior without forcing all current V0 call sites to change immediately.

Recommended approach:

- keep the existing `ValidateConfig(...)` entrypoint as a legacy wrapper
- add a new `ValidateConfigWithOptions(...)` entrypoint for future V1 callers
- make the legacy wrapper pass an explicit V0 alias preset instead of relying on implicit field-name fallback

This removes ambiguity about how V0 input continues to resolve and makes the no-regression behavior part of the API contract.

### D-7: Do Not Strengthen Existing Validation Semantics in This Change

This work is about version-aware key handling and custom-type-ref recognition. It should not silently tighten unrelated config validation behavior.

The following existing shared-validator semantics must remain unchanged:

1. `enum` validation is shallow:
   - value must be an array
   - duplicate entries are rejected
   - enum element types are **not** validated against the enclosing config type
2. `pattern` validation is shallow:
   - value must be a string
   - the string is **not** compiled as a regex in this validator layer
3. Unknown or invalid type names remain the responsibility of upstream syntax validation:
   - if no validator can be constructed for a type, the shared config validator should continue to defer rather than inventing a new type-level error

This is important because the current tests explicitly permit mixed-type enum arrays, and no current validator compiles regex patterns.

### D-8: Validators Accept Both Raw Key and Keyword

The `TypeConfigValidator` interface should change to accept both the user's raw input key and the resolved `ConfigKeyword`:

```go
ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error)
ValidateCrossFields(config map[ConfigKeyword]any) []rules.ValidationResult
```

Validators use `keyword` for switch-case matching and allowed-key lookup, and `rawKey` for constructing `Reference` and `Message` fields in `ValidationResult`.

This ensures:

1. Error messages say `"'min_length' must be an integer"` (raw) not `"'minLength' must be an integer"` (canonical V0) when called from V1
2. Error references point to `"min_length"` (raw) not `"min_length"` (keyword string value)
3. No post-processing denormalization step is needed — correctness is guaranteed at the source
4. The blast radius is contained to the `config` package (~10 validator implementations) plus one external override (`customTypeObjectConfig` in `customtype_config_valid.go`)

The `TypeConfigValidator` interface is package-internal; the public API (`ValidateConfig` / `ValidateConfigWithOptions`) does not change. External callers are unaffected.

For cross-field validation, the method receives a `map[ConfigKeyword]any` built from resolved fields only. Since `ConfigKeyword` is `type ConfigKeyword string`, cross-field error messages naturally use keyword string values (e.g., `"min_length cannot be greater than max_length"`). This is a minor cosmetic change for V0 cross-field messages (previously `"minLength cannot be greater than maxLength"`) which is acceptable because cross-field errors reference the config root, not individual fields.

---

## 4. Proposed Architecture

### 4.1 New Options Framework

Introduce an options-driven validation entrypoint inside `cli/internal/providers/datacatalog/rules/config/`.

```go
type ConfigKeyword string

const (
	KeywordEnum                 ConfigKeyword = "enum"
	KeywordMinimum              ConfigKeyword = "minimum"
	KeywordMaximum              ConfigKeyword = "maximum"
	KeywordPattern              ConfigKeyword = "pattern"
	KeywordFormat               ConfigKeyword = "format"
	KeywordMinLength            ConfigKeyword = "min_length"
	KeywordMaxLength            ConfigKeyword = "max_length"
	KeywordExclusiveMinimum     ConfigKeyword = "exclusive_minimum"
	KeywordExclusiveMaximum     ConfigKeyword = "exclusive_maximum"
	KeywordMultipleOf           ConfigKeyword = "multiple_of"
	KeywordItemTypes            ConfigKeyword = "item_types"
	KeywordMinItems             ConfigKeyword = "min_items"
	KeywordMaxItems             ConfigKeyword = "max_items"
	KeywordUniqueItems          ConfigKeyword = "unique_items"
	KeywordAdditionalProperties ConfigKeyword = "additional_properties"
)

type ValidateConfigOption func(*validateConfigOptions)

func WithFieldAliases(aliases map[string]ConfigKeyword) ValidateConfigOption
func WithCustomTypeRefMatcher(fn func(string) bool) ValidateConfigOption
func WithValidatorOverrides(overrides map[string]TypeConfigValidator) ValidateConfigOption
```

Notes:

- `WithFieldAliases` configures accepted raw-to-keyword mappings such as `min_length -> KeywordMinLength`
- `WithCustomTypeRefMatcher` makes custom type handling version-aware without exposing regex assumptions in the validator flow
- `WithValidatorOverrides` preserves the current custom-type object override behavior in the same options framework
- alias presets should be explicit and complete for each supported input shape rather than relying on hidden default keyword lookup

The options framework carries behavior only. It does not supply custom error messages.

### 4.2 Entry Point Strategy

Retain the current public behavior for V0 callers and add a new options-aware path for future V1 use.

```go
func ValidateConfig(
	types []string,
	config map[string]any,
	reference string,
	validatorOverrides map[string]TypeConfigValidator,
) []rules.ValidationResult {
	return ValidateConfigWithOptions(
		types,
		config,
		reference,
		WithFieldAliases(v0FieldAliases),
		WithValidatorOverrides(validatorOverrides),
		WithCustomTypeRefMatcher(legacyCustomTypeRefMatcher),
	)
}

func ValidateConfigWithOptions(
	types []string,
	config map[string]any,
	reference string,
	opts ...ValidateConfigOption,
) []rules.ValidationResult
```

This keeps current V0 consumers stable and gives future V1 rule handlers an explicit API to opt into V1 behavior.

Recommended presets:

```go
var v0FieldAliases = map[string]ConfigKeyword{
	"enum":                 KeywordEnum,
	"minimum":              KeywordMinimum,
	"maximum":              KeywordMaximum,
	"pattern":              KeywordPattern,
	"format":               KeywordFormat,
	"minLength":            KeywordMinLength,
	"maxLength":            KeywordMaxLength,
	"exclusiveMinimum":     KeywordExclusiveMinimum,
	"exclusiveMaximum":     KeywordExclusiveMaximum,
	"multipleOf":           KeywordMultipleOf,
	"itemTypes":            KeywordItemTypes,
	"minItems":             KeywordMinItems,
	"maxItems":             KeywordMaxItems,
	"uniqueItems":          KeywordUniqueItems,
	"additionalProperties": KeywordAdditionalProperties,
}

var v1FieldAliases = map[string]ConfigKeyword{
	"enum":                  KeywordEnum,
	"minimum":               KeywordMinimum,
	"maximum":               KeywordMaximum,
	"pattern":               KeywordPattern,
	"format":                KeywordFormat,
	"min_length":            KeywordMinLength,
	"max_length":            KeywordMaxLength,
	"exclusive_minimum":     KeywordExclusiveMinimum,
	"exclusive_maximum":     KeywordExclusiveMaximum,
	"multiple_of":           KeywordMultipleOf,
	"item_types":            KeywordItemTypes,
	"min_items":             KeywordMinItems,
	"max_items":             KeywordMaxItems,
	"unique_items":          KeywordUniqueItems,
	"additional_properties": KeywordAdditionalProperties,
}
```

The shared package may expose these as reusable presets, or callers may compose them locally. The important part is that V0 compatibility remains explicit and testable.

### 4.3 Normalization and Validation Flow

The normalization layer resolves alias mappings and distributes fields into the correct validation paths. There is no post-processing denormalization step — raw key preservation is handled at the source by passing `rawKey` into validators.

**Step 1: Resolve aliases**

1. Iterate over raw config keys
2. Look up each raw key in the effective alias map supplied in options
3. For resolved keys: record `(rawKey, keyword, value)` for field-level validation, and add `keyword -> value` to the cross-field map
4. For unresolved keys: record `(rawKey, value)` for the unresolved field path

Suggested shape:

```go
type resolvedField struct {
	RawKey  string
	Keyword ConfigKeyword
	Value   any
}

func resolveAliases(
	config map[string]any,
	aliases map[string]ConfigKeyword,
) (resolved []resolvedField, crossFieldMap map[ConfigKeyword]any, unresolved map[string]any)
```

**Step 2: Field-level validation**

5. For each resolved field: call `validateFieldUnion(validators, rawKey, keyword, value, reference)` — validators use `keyword` for matching and `rawKey` in error output
6. For each unresolved field: call `validateFieldUnion(validators, rawKey, ConfigKeyword(""), value, reference)` — no validator's allowed-key map will match an empty keyword, so all validators return `ErrFieldNotSupported`, producing the existing "not applicable for type(s)" message

**Step 3: Cross-field validation**

7. Run cross-field validation against the `map[ConfigKeyword]any` containing only resolved fields
8. Cross-field validators look up keywords directly (e.g., `config[KeywordMinLength]`)

Important:

- error `Reference` and `Message` fields use `rawKey` at the source — no post-processing rewrite is needed
- cross-field errors continue to point at the config object root, as they do today
- cross-field error messages use keyword string values (e.g., `"min_length cannot be greater than max_length"`) which is a minor cosmetic change from the current V0 camelCase messages
- unresolved keys are not inserted into the cross-field map

### 4.4 Validator Interface Change and Options-Aware Construction

#### Interface Change

The `TypeConfigValidator` interface must be updated to accept both the raw key and keyword (see D-8):

```go
type TypeConfigValidator interface {
	ConfigAllowed() bool
	ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error)
	ValidateCrossFields(config map[ConfigKeyword]any) []rules.ValidationResult
}
```

Each type validator must update:

- **Allowed-key maps**: change from `map[string]bool` to `map[ConfigKeyword]bool`
- **`ValidateField` switch cases**: match on `keyword` (e.g., `case KeywordMinLength, KeywordMaxLength:`) instead of raw strings
- **`ValidateField` error output**: use `rawKey` in `Reference` and `Message` fields
- **`ValidateCrossFields` key lookups**: use keyword constants (e.g., `config[KeywordMinLength]`) instead of camelCase strings (e.g., `config["minLength"]`)

Example — `StringTypeConfig` after the change:

```go
var allowedStringKeys = map[ConfigKeyword]bool{
	KeywordEnum:      true,
	KeywordMinLength: true,
	KeywordMaxLength: true,
	KeywordPattern:   true,
	KeywordFormat:    true,
}

func (s *StringTypeConfig) ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedStringKeys[keyword] {
		return nil, ErrFieldNotSupported
	}

	switch keyword {
	case KeywordEnum:
		return validateEnum(rawKey, fieldval)
	case KeywordMinLength, KeywordMaxLength:
		if !isInteger(fieldval) {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be an integer", rawKey),
			}}, nil
		}
		// ...
	}
	return nil, nil
}

func (s *StringTypeConfig) ValidateCrossFields(config map[ConfigKeyword]any) []rules.ValidationResult {
	minLength, hasMin := config[KeywordMinLength]
	maxLength, hasMax := config[KeywordMaxLength]
	// ...
}
```

#### Validators That Must Update

| Validator | Field-level change | Cross-field change |
|-----------|-------------------|-------------------|
| `StringTypeConfig` | allowed keys + switch + messages | `KeywordMinLength` / `KeywordMaxLength` lookup |
| `IntegerTypeConfig` | allowed keys + switch + messages | `KeywordMinimum` / `KeywordMaximum` + `KeywordExclusiveMinimum` / `KeywordExclusiveMaximum` lookup |
| `NumberTypeConfig` | allowed keys + switch + messages | Same as integer |
| `ArrayTypeConfig` | allowed keys + switch + messages | `KeywordMinItems` / `KeywordMaxItems` lookup |
| `BooleanTypeConfig` | allowed keys + switch + messages | None (no cross-field) |
| `ObjectTypeConfig` | Trivial (returns `ErrFieldNotSupported`) | None |
| `NullTypeConfig` | Trivial (returns `ErrFieldNotSupported`) | None |
| `CustomTypeConfig` | Trivial (returns `ErrFieldNotSupported`) | None |
| `customTypeObjectConfig` (external) | allowed keys + switch + messages | None |

#### Options-Aware Type Resolution

Type resolution must stop depending on the legacy regex directly:

```go
func getDefaultValidatorForType(typeName string, opts validateConfigOptions) TypeConfigValidator {
	switch typeName {
	case "string":
		return &StringTypeConfig{}
	case "integer":
		return &IntegerTypeConfig{}
	case "number":
		return &NumberTypeConfig{}
	case "array":
		return &ArrayTypeConfig{isCustomTypeRef: opts.customTypeRefMatcher}
	case "object":
		return &ObjectTypeConfig{}
	case "boolean":
		return &BooleanTypeConfig{}
	case "null":
		return &NullTypeConfig{}
	default:
		if opts.customTypeRefMatcher != nil && opts.customTypeRefMatcher(typeName) {
			return &CustomTypeConfig{}
		}
		return nil
	}
}
```

The array validator must also use the configured matcher for `itemTypes` custom type reference checks:

```go
type ArrayTypeConfig struct {
	isCustomTypeRef func(string) bool
}
```

---

## 5. V1 Option Set Behavior

### 5.1 Accepted V0 Aliases

The legacy `ValidateConfig(...)` wrapper must resolve V0 camelCase input through an explicit alias preset.

| Raw V0 key | Config keyword |
|------------|----------------|
| `enum` | `KeywordEnum` |
| `minimum` | `KeywordMinimum` |
| `maximum` | `KeywordMaximum` |
| `pattern` | `KeywordPattern` |
| `format` | `KeywordFormat` |
| `minLength` | `KeywordMinLength` |
| `maxLength` | `KeywordMaxLength` |
| `exclusiveMinimum` | `KeywordExclusiveMinimum` |
| `exclusiveMaximum` | `KeywordExclusiveMaximum` |
| `multipleOf` | `KeywordMultipleOf` |
| `itemTypes` | `KeywordItemTypes` |
| `minItems` | `KeywordMinItems` |
| `maxItems` | `KeywordMaxItems` |
| `uniqueItems` | `KeywordUniqueItems` |
| `additionalProperties` | `KeywordAdditionalProperties` |

This alias preset should be passed automatically by the legacy wrapper so existing V0 callers do not need to change.

### 5.2 Accepted V1 Aliases

The V1 configuration passed through `ValidateConfigWithOptions(...)` must accept the following raw-to-keyword mappings:

| Raw V1 key | Config keyword |
|------------|----------------|
| `enum` | `KeywordEnum` |
| `minimum` | `KeywordMinimum` |
| `maximum` | `KeywordMaximum` |
| `pattern` | `KeywordPattern` |
| `format` | `KeywordFormat` |
| `min_length` | `KeywordMinLength` |
| `max_length` | `KeywordMaxLength` |
| `exclusive_minimum` | `KeywordExclusiveMinimum` |
| `exclusive_maximum` | `KeywordExclusiveMaximum` |
| `multiple_of` | `KeywordMultipleOf` |
| `item_types` | `KeywordItemTypes` |
| `min_items` | `KeywordMinItems` |
| `max_items` | `KeywordMaxItems` |
| `unique_items` | `KeywordUniqueItems` |
| `additional_properties` | `KeywordAdditionalProperties` |

The V1 alias preset should be explicit and complete. This keeps the resolution model symmetrical with V0 and avoids implicit fallback behavior that is easy to misread or regress.

### 5.3 Unknown V1 Keys

If a raw V1 key is not present in the supplied alias map, it should continue through the existing unknown-key / not-applicable behavior.

Examples:

- `minLength` is not handled specially in the shared package
- if V1 aliases only include `min_length`, then `minLength` should fall through and be handled as an unknown / not-applicable key
- unknown keys must not gain a new dedicated message path in this work

### 5.4 Mixed Case Input

If both a valid snake_case key and an unaliased camelCase key are present in the same V1 config object:

- the snake_case key is validated normally
- the unaliased camelCase key flows through the existing unknown-key / not-applicable behavior

No special duplicate/conflict handling is required for this case.

---

## 6. Custom Type Object Override

The existing custom type object override must remain, but its V1-facing spelling changes through the aliasing layer.

Desired behavior:

- V0 custom type object config continues to accept `additionalProperties`
- V1 custom type object config accepts only `additional_properties`
- V1 `additionalProperties` is not aliased and therefore falls through the existing unknown-key / not-applicable behavior
- any other object config key continues to be rejected by the existing override semantics

This is achieved by mapping accepted V1 `additional_properties` to the shared validator keyword for additional properties.

### Override Contract with Normalization

Validator overrides receive `rawKey` and `keyword` through the same interface as default validators, since normalization resolves the keyword before any validator is invoked.

The `customTypeObjectConfig` override must update to match on `KeywordAdditionalProperties` instead of the raw string `"additionalProperties"`:

```go
var allowedCustomTypeObjectKeys = map[ConfigKeyword]bool{
	KeywordAdditionalProperties: true,
}

func (c *customTypeObjectConfig) ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedCustomTypeObjectKeys[keyword] {
		return nil, config.ErrFieldNotSupported
	}
	switch keyword {
	case KeywordAdditionalProperties:
		if _, ok := fieldval.(bool); !ok {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be a boolean", rawKey),
			}}, nil
		}
	}
	return nil, nil
}
```

Resolution flow for the override:

- V0: raw `"additionalProperties"` -> alias -> `KeywordAdditionalProperties` -> override matches, uses `"additionalProperties"` in messages
- V1: raw `"additional_properties"` -> alias -> `KeywordAdditionalProperties` -> override matches, uses `"additional_properties"` in messages
- V1 wrong-casing: raw `"additionalProperties"` -> not in V1 alias map -> unresolved, empty keyword -> override's allowed-key map rejects -> `ErrFieldNotSupported` -> "not applicable for type(s)"

**Note on `additionalProperties` in the V0 preset**: The V0 alias preset maps `"additionalProperties" -> KeywordAdditionalProperties`. No default type validator in the `config` package recognizes this keyword — it is consumed only via the `customTypeObjectConfig` override. For non-custom-type callers, `additionalProperties` will correctly fall through to "not applicable for type(s)" via union semantics.

---

## 7. Implementation Guidance

### Recommended Files to Add or Update

Likely changes for this work:

```text
cli/internal/providers/datacatalog/rules/config/
├── validator.go                  # new options-aware entrypoint, normalization flow, updated validateFieldUnion
├── utils.go                      # remove direct legacy-only matcher dependency
├── string_validator.go           # keyword-aware allowed keys, rawKey in messages, keyword cross-field lookups
├── integer_validator.go          # keyword-aware allowed keys, rawKey in messages, keyword cross-field lookups
├── number_validator.go           # keyword-aware allowed keys, rawKey in messages, keyword cross-field lookups
├── array_validator.go            # keyword-aware + options-aware custom type ref checks, keyword cross-field lookups
├── boolean_validator.go          # keyword-aware allowed keys, rawKey in messages
├── object_validator.go           # trivial interface update (returns ErrFieldNotSupported)
├── null_validator.go             # trivial interface update (returns ErrFieldNotSupported)
├── customtype_validator.go       # trivial interface update, type resolution path changes
├── options.go                    # new functional options plus explicit V0/V1 alias presets
└── validator_test.go             # new V1 and backward-compat coverage

cli/internal/providers/datacatalog/rules/customtype/
└── customtype_config_valid.go    # customTypeObjectConfig override: keyword-aware interface update
```

### Recommended Non-Changes

The following should not be expanded in this PR:

1. property V1 rule wiring
2. custom type V1 rule wiring
3. semantic validation
4. top-level property `item_type` / `item_types` validation

### Future Usage Example

This is not part of the current implementation, but the API should make later wiring simple:

```go
results := config.ValidateConfigWithOptions(
	types,
	property.Config,
	reference,
	config.WithFieldAliases(v1FieldAliases),
	config.WithCustomTypeRefMatcher(v1CustomTypeRefMatcher),
)
```

---

## 8. Edge Cases That Must Be Covered

The implementation must preserve the following behavior:

1. Empty config remains valid
2. V0 config behavior remains unchanged
3. V0 compatibility does not rely on implicit raw-key fallback; it is driven through an explicit V0 alias preset
4. Cross-field validation still uses normalized canonical keys
5. Deduplication behavior remains unchanged
6. V1 accepts unchanged keys such as `enum`, `minimum`, `maximum`, `pattern`, and `format`
7. V1 introduces no special wrong-spelling path in the shared package
8. V1 legacy custom type refs inside `item_types` may fail through the normal invalid-type path
9. V1 object config accepts only `additional_properties`
10. V1 unaliased camelCase `additionalProperties` falls through the existing unknown-key / not-applicable behavior
11. Raw user-facing references in field errors must use the original input key, not the normalized `ConfigKeyword`
12. `enum` remains array-and-duplicate validated only; enum element types are not newly type-checked in this work
13. `pattern` remains string-validated only; regex compilation is not introduced in this work
14. If no validator can be constructed for a type, shared config validation still defers to the surrounding spec/type syntax validators
15. If `WithFieldAliases` receives an empty map, all config keys are treated as unresolved and produce the existing "not applicable for type(s)" behavior
16. If `ValidateConfigWithOptions` is called with zero options (no aliases, no matcher, no overrides), it must not panic; behavior should be equivalent to passing empty aliases
17. A V1 config containing both `"min_length": 5` (resolved) and `"minLength": 10` (unresolved) validates `min_length` normally and produces "not applicable for type(s)" for `minLength`
18. A misconfigured custom type ref matcher that returns `true` for a primitive type name (e.g., `"string"`) must not override the built-in type validator, because `getDefaultValidatorForType` checks built-in types before the custom type ref matcher
19. V0 cross-field error messages may change from camelCase (e.g., `"minLength cannot be greater than maxLength"`) to keyword form (e.g., `"min_length cannot be greater than max_length"`); this is an acceptable cosmetic change

---

## 9. Test Strategy

### Unit Tests

The implementation must add table-driven tests covering at least:

1. V0 wrapper behavior remains unchanged
2. Legacy wrapper passes an explicit V0 alias preset and does not rely on implicit raw-key fallback
3. V1 normalization accepts valid snake_case keys
4. V1 unchanged-key acceptance for `enum`, `minimum`, `maximum`, `pattern`, `format`
5. Unaliased camelCase keys fall through the existing unknown-key / not-applicable behavior
6. V1 array `item_types` accepts primitive types
7. V1 array `item_types` accepts `#custom-types:<id>`
8. V1 array `item_types` rejects invalid references through the normal invalid-type path
9. V1 custom type object config accepts `additional_properties`
10. V1 custom type object config with unaliased `additionalProperties` falls through the existing unknown-key / not-applicable behavior
11. Mixed valid snake_case + unaliased camelCase in the same config map
12. Cross-field validation still works after normalization:
    - `min_length > max_length`
    - `min_items > max_items`
    - `exclusive_minimum >= exclusive_maximum`
13. Validator override behavior still works with the options-aware entrypoint
14. V1 top-level custom type type detection works in the shared package:
    - `ValidateConfigWithOptions([]string{"#custom-types:Address"}, ...)` rejects config as not allowed
15. `enum` mixed element types remain valid where they are valid today
16. `pattern` invalid regex syntax is not rejected solely for being an invalid regex pattern
17. Unknown type names still cause the shared config validator to defer rather than producing a new type-level error
18. Empty alias map: all fields produce "not applicable for type(s)"
19. Zero options: `ValidateConfigWithOptions` called with no options does not panic
20. Dual-cased keys in V1: `{"min_length": 5, "minLength": 10}` — first validates, second produces "not applicable"
21. Custom type ref matcher returning `true` for `"string"` — built-in string validator takes precedence
22. Error `Reference` uses raw key: V1 `"min_length"` field errors reference `"min_length"` not `"minLength"` in results
23. Error `Message` uses raw key: V1 error says `"'min_length' must be an integer"` not `"'minLength' must be an integer"`
24. Cross-field with V1 input: `{"min_length": 10, "max_length": 5}` produces min > max error using keyword string values

### Suggested Coverage Focus

The highest-value coverage should be on:

- keyword alias resolution logic
- options-aware validation flow
- array custom type reference handling
- custom type object override behavior

### Verification Commands

The implementation must be validated with:

```bash
go build ./...
make test
make lint
```

If the implementation adds or changes package-level tests, a focused coverage run for the changed package is also recommended:

```bash
go test ./cli/internal/providers/datacatalog/rules/config/... -cover
```

---

## 10. Acceptance Criteria

### Functional

- [ ] A focused V1-aware config validation path exists in `cli/internal/providers/datacatalog/rules/config/`
- [ ] The existing `ValidateConfig(...)` function remains usable for legacy callers and preserves current V0 behavior
- [ ] The legacy `ValidateConfig(...)` wrapper passes an explicit V0 field-alias preset
- [ ] A new `ValidateConfigWithOptions(...)` entrypoint exists for future V1 rule wiring
- [ ] `WithFieldAliases(...)` uses `ConfigKeyword` values rather than raw internal string names
- [ ] V0 camelCase config keys are resolved through explicit aliases rather than implicit fallback
- [ ] V1 accepts snake_case config keys and normalizes them to shared validator keywords
- [ ] V1 does not introduce a dedicated wrong-spelling error path in the shared package
- [ ] V1 array custom type references are recognized only in `#custom-types:<id>` format
- [ ] Legacy custom type refs inside V1 `item_types` are allowed to fail through the normal invalid-type path
- [ ] V1 custom type object config accepts only `additional_properties`
- [ ] V1 `additionalProperties` is not aliased and therefore falls through the existing unknown-key / not-applicable behavior
- [ ] Raw field references in validation results use the original input key provided by the user
- [ ] Error messages in validation results use the original input key provided by the user
- [ ] `TypeConfigValidator.ValidateField` accepts both `rawKey string` and `keyword ConfigKeyword`
- [ ] `TypeConfigValidator.ValidateCrossFields` accepts `map[ConfigKeyword]any`
- [ ] All type validators in the `config` package are updated to the new interface
- [ ] The `customTypeObjectConfig` override in `customtype_config_valid.go` is updated to the new interface
- [ ] Existing union semantics, cross-field semantics, and deduplication behavior remain unchanged
- [ ] Existing shallow `enum` behavior remains unchanged
- [ ] Existing shallow `pattern` behavior remains unchanged
- [ ] Existing deferral behavior for unknown/invalid types remains unchanged
- [ ] `ValidateConfigWithOptions` with zero options does not panic
- [ ] `ValidateConfigWithOptions` with empty alias map treats all fields as unresolved

### Scope

- [ ] The implementation is limited to shared config validation logic
- [ ] No V1 semantic validation is introduced in this work
- [ ] No property/custom-type V1 rule wiring is introduced in this work
- [ ] No validation for top-level property `item_type` / `item_types` is introduced in this work

### Quality

- [ ] Test coverage for changed files or changed packages is greater than 85%
- [ ] New behavior is covered by table-driven unit tests
- [ ] `go build ./...` passes after the implementation is complete
- [ ] `make test` passes after the implementation is complete
- [ ] `make lint` passes after the implementation is complete

---

## 11. Risks and Follow-Ups

### Risks

1. If raw-key references are not preserved, V1 error paths will point to canonical camelCase names instead of the user input
2. If custom type reference matching remains hard-coded to the legacy regex anywhere in the shared path, V1 arrays will silently miss valid custom type refs
3. If the current `ValidateConfig` signature is replaced instead of wrapped, later rule-wiring deferral will cause unnecessary churn in existing V0 callers
4. If V0 compatibility depends on implicit fallback rather than an explicit alias preset, the `ConfigKeyword` abstraction is likely to be misread and regress later
5. The `TypeConfigValidator` interface changes (`ValidateField` gains `rawKey` + `keyword`, `ValidateCrossFields` gains `map[ConfigKeyword]any`). While the blast radius is contained to the `config` package and one external override (`customTypeObjectConfig`), all ~10 implementations must be updated mechanically. Any out-of-tree override will break at compile time.
6. V0 cross-field error messages will change from camelCase (e.g., `"minLength cannot be greater than maxLength"`) to keyword string values (e.g., `"min_length cannot be greater than max_length"`). This is a cosmetic change that may affect snapshot-based test assertions in V0 callers.

### Follow-Ups

1. Wire V1 property config validation to the new options-aware API
2. Wire V1 custom type config validation to the new options-aware API
3. Add V1 semantic validation for config references in follow-up PRs
4. Review V0-to-V1 migration helpers separately to ensure config-key conversion remains aligned with the chosen V1 spellings
