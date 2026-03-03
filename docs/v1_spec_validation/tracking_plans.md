# Tracking Plans - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **exist** in the new validation engine. Two rules are registered via `LegacyVersionPatterns("tp")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `datacatalog/tracking-plans/spec-syntax-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/trackingplan/trackingplan_spec_valid.go` |
| `datacatalog/tracking-plans/semantic-valid` | Semantic | `cli/internal/providers/datacatalog/rules/trackingplan/trackingplan_semantic_valid.go` |

These rules decode into V0 structs (`TrackingPlan` / `TPRule`) and match `kind: tp` only. They do **not** match V1 specs which use `kind: tracking-plan` and `version: rudder/v1`.

**Important**: Tracking plans have a **different Kind** between V0.1 and V1:
- V0.1: `kind: tp`
- V1: `kind: tracking-plan`

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`TrackingPlan` / `TPRule`) | V1 (`TrackingPlanV1` / `TPRuleV1`) |
|--------|-----------------------------------|-------------------------------------|
| Kind string | `tp` | `tracking-plan` |
| Event reference | Object `TPRuleEvent` with `$ref` field | Direct string field `Event` (e.g. `#event:<id>`) |
| Property reference | `TPRuleProperty` with `$ref` field | `TPRulePropertyV1` with `property` field |
| Unplanned handling | `AllowUnplanned` on `TPRuleEvent` | `AdditionalProperties` on `TPRuleV1` and `TPRulePropertyV1` |
| Variant property refs | `PropertyReference` with `$ref` | `PropertyReferenceV1` with `property` |
| Include ref format | `#/tp/<group>/event_rule/<id>` | Same format |
| Validation tags | Has `validate:"required"`, `validate:"pattern=display_name"` etc. | No validation tags |

### V0.1 Structs (`TrackingPlan`, `TPRule`, `TPRuleProperty` in `localcatalog/tracking_plan.go`)

```go
type TrackingPlan struct {
    Name        string    `json:"display_name" validate:"required,pattern=display_name"`
    LocalID     string    `json:"id" validate:"required"`
    Description string    `json:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    Rules       []*TPRule `json:"rules,omitempty" validate:"omitempty,dive"`
}

type TPRule struct {
    Type       string            `json:"type"`
    LocalID    string            `json:"id" validate:"required"`
    Event      *TPRuleEvent      `json:"event,omitempty"`
    Properties []*TPRuleProperty `json:"properties,omitempty" validate:"omitempty,dive"`
    Includes   *TPRuleIncludes   `json:"includes,omitempty"`
    Variants   Variants          `json:"variants,omitempty" validate:"omitempty,max=1,dive"`
}

type TPRuleEvent struct {
    Ref             string `json:"$ref" validate:"required,pattern=legacy_event_ref"`
    AllowUnplanned  bool   `json:"allow_unplanned"`
    IdentitySection string `json:"identity_section"`
}

type TPRuleProperty struct {
    Ref             string            `json:"$ref" validate:"required,pattern=legacy_property_ref"`
    Required        bool              `json:"required"`
    AllowUnplanned  *bool             `json:"allow_unplanned,omitempty"`
    Properties      []*TPRuleProperty `json:"properties,omitempty" validate:"omitempty,dive"`
}
```

### V1 Structs (`TrackingPlanV1`, `TPRuleV1`, `TPRulePropertyV1` in `localcatalog/tracking_plan.go`)

```go
type TrackingPlanV1 struct {
    Name        string      `json:"display_name"`
    LocalID     string      `json:"id"`
    Description string      `json:"description,omitempty"`
    Rules       []*TPRuleV1 `json:"rules,omitempty"`
}

type TPRuleV1 struct {
    Type                 string              `json:"type"`
    LocalID              string              `json:"id"`
    Event                string              `json:"event"`
    IdentitySection      string              `json:"identity_section,omitempty"`
    AdditionalProperties bool                `json:"additionalProperties,omitempty"`
    Properties           []*TPRulePropertyV1 `json:"properties,omitempty"`
    Includes             *TPRuleIncludes     `json:"includes,omitempty"`
    Variants             VariantsV1          `json:"variants,omitempty"`
}

type TPRulePropertyV1 struct {
    Property             string              `json:"property"`
    Required             bool                `json:"required"`
    AdditionalProperties *bool               `json:"additionalProperties,omitempty"`
    Properties           []*TPRulePropertyV1 `json:"properties,omitempty"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("tracking-plan", "rudder/v1")` and decode into `TrackingPlanV1` / `TPRuleV1`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `display_name` required | Tracking plan must have a non-empty `display_name` field |
| 2 | Rule `id` required | Each rule must have a non-empty `id` field |
| 3 | Rule must have `event` or `includes` (not both, not neither) | Each rule must specify either an `event` string ref or an `includes` block, but not both |
| 4 | Properties without events not allowed | Rules with `properties` must also have `event` (not just `includes`) |
| 5 | Variant structure validation | Max 1 variant per rule; variant `type` must be `"discriminator"`; `discriminator` required; `cases` min 1 element; each case needs `display_name`, `match` (min 1, values string/bool/integer), `properties` (min 1, each with `property` field); default properties must have `property` field |
| 6 | Nesting depth <= 3 levels | Nested properties within rules must not exceed 3 levels of nesting depth |
| 7 | `additionalProperties` only for nested properties | The `additionalProperties` field on `TPRulePropertyV1` is only valid for properties that have nested sub-properties |
| 8 | Nested property `property` field required, format valid | Each nested property must have a non-empty `property` field in the V1 ref format (`#property:<id>`) |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Event refs resolve | The `event` string (format `#event:<id>`) must resolve to an existing event in the graph |
| 2 | Property refs resolve | Each `property` field in rule properties (format `#property:<id>`) must resolve to an existing property in the graph |
| 3 | Include refs resolve | Include references (format `#/tp/<group>/event_rule/<id>`) must resolve to existing tracking plan event rules |
| 4 | Nested property refs resolve and parent allows nesting | Nested property refs must resolve, and the parent property must be of type `object` or `array` with item type `object` |
| 5 | Variant discriminator refs resolve, type constraint | Discriminator (format `#property:<id>`) must resolve to a property with type string, integer, or boolean; discriminator must be one of the rule's own properties |
| 6 | `display_name` uniqueness | No two tracking plans may share the same display_name |

Note: Rule `id` uniqueness within a TP and `id` uniqueness across TPs are handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] All 8 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("tracking-plan", "rudder/v1")` and decoding into `TrackingPlanV1`
- [ ] All 6 semantic validations listed above are implemented as V1 rules
- [ ] Event refs use V1 format (string `#event:<id>` instead of object with `$ref`)
- [ ] Property refs use V1 format (`property` field with `#property:<id>` instead of `$ref`)
- [ ] `additionalProperties` is validated instead of `allow_unplanned`
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-tracking-plans-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `tracking-plan` resource in the datacatalog provider. These rules target `rudder/v1` specs with `kind: tracking-plan` (distinct from V0.1's `kind: tp`) and decode into `TrackingPlanV1`/`TPRuleV1` structs, implementing 8 syntactic and 6 semantic validations.

---

## Changes

* Add V1 syntactic rule for tracking plan spec validation (required fields, event/includes exclusivity, variant structure, nesting depth, additionalProperties)
* Add V1 semantic rule for event/property/include reference resolution, variant discriminator validation, and display_name uniqueness

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 tracking plan specs

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
