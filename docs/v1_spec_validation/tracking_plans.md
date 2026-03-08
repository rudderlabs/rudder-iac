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

**Variants**: Variant validation is handled separately (e.g. in custom types). Tracking plan V1 validation must **not** perform any variant validation—do not call any function to validate variants on `TrackingPlanV1` or its rules.

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`TrackingPlan` / `TPRule`) | V1 (`TrackingPlanV1` / `TPRuleV1`) |
|--------|-----------------------------------|-------------------------------------|
| Kind string | `tp` | `tracking-plan` |
| Event reference | Object `TPRuleEvent` with `$ref` field | Direct string field `Event` (e.g. `#events:<id>`), required |
| Property reference | `TPRuleProperty` with `$ref` field | `TPRulePropertyV1` with `property` field |
| Unplanned handling | `AllowUnplanned` on `TPRuleEvent` | `AdditionalProperties` on `TPRuleV1` and `TPRulePropertyV1` |
| Validation tags | Has `validate:"required"`, `validate:"pattern=display_name"` etc. | **Add matching tags** (see below) |

V1 syntactic and semantic rules must use **`V1VersionPatterns`** (e.g. `prules.V1VersionPatterns(localcatalog.KindTrackingPlansV1)`) when registering the rule, not `LegacyVersionPatterns`.

### V0.1 Structs (`TrackingPlan`, `TPRule`, `TPRuleProperty` in `localcatalog/tracking_plan.go`)

```go
type TrackingPlan struct {
    Name        string    `json:"display_name" validate:"required,pattern=display_name"`
    LocalID     string    `json:"id" validate:"required"`
    Description string    `json:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    Rules       []*TPRule `json:"rules,omitempty" validate:"omitempty,dive"`
}

type TPRule struct {
    Type       string            `json:"type" validate:"required,eq=event_rule"`
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

### V1 Structs — Current (`TrackingPlanV1`, `TPRuleV1`, `TPRulePropertyV1` in `localcatalog/tracking_plan.go`)

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

### V1 Structs — Updated (tags to add)

- **Event** on `TPRuleV1`: must be a valid event reference via the `event_ref` tag (pattern `#events:<id>`). Event is **required**; this enforces that rules with properties must have an event (no custom logic needed for "properties without event").
- **Type** on both `TPRule` (V0) and `TPRuleV1` (V1): required and must equal `"event_rule"` via go-validator tag `validate:"required,eq=event_rule"`.
- **identity_section** on `TPRuleV1`: same tags as V0 (`TPRuleEvent.IdentitySection`): `validate:"omitempty,oneof=properties traits context.traits"`.
- **Variants**: Do not add validation tags for variants on tracking plan; variant validation is handled separately.
- **Includes**: Includes is an old concept and will be removed; do not add a specific tag or custom logic for it.

```go
type TrackingPlanV1 struct {
    Name        string      `json:"display_name" validate:"required,pattern=display_name"`
    LocalID     string      `json:"id" validate:"required"`
    Description string      `json:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    Rules       []*TPRuleV1 `json:"rules,omitempty" validate:"omitempty,dive"`
}

type TPRuleV1 struct {
    Type                 string              `json:"type" validate:"required,eq=event_rule"`
    LocalID              string              `json:"id" validate:"required"`
    Event                string              `json:"event" validate:"required,pattern=event_ref"`
    IdentitySection      string              `json:"identity_section,omitempty" validate:"omitempty,oneof=properties traits context.traits"`
    AdditionalProperties bool                `json:"additionalProperties,omitempty"`
    Properties           []*TPRulePropertyV1 `json:"properties,omitempty" validate:"omitempty,dive"`
    Includes             *TPRuleIncludes     `json:"includes,omitempty"`
    Variants             VariantsV1          `json:"variants,omitempty"`
}

type TPRulePropertyV1 struct {
    Property             string              `json:"property" validate:"required,pattern=property_ref"`
    Required             bool                `json:"required"`
    AdditionalProperties *bool               `json:"additionalProperties,omitempty"`
    Properties           []*TPRulePropertyV1 `json:"properties,omitempty" validate:"omitempty,dive"`
}
```

The `display_name`, `letter_start`, `property_ref`, and `event_ref` patterns are already registered.

---

## Syntactic Validations to Add for V1

V1 syntactic rule must use **`V1VersionPatterns(localcatalog.KindTrackingPlansV1)`** so it targets `kind: tracking-plan` with `version: rudder/v1`, and must decode into `TrackingPlanV1` / `TPRuleV1`. Do not validate variants in this rule (variants are handled separately).

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `display_name` required + pattern | `validate:"required,pattern=display_name"` on `Name` | Tracking plan must have a valid display name |
| 2 | `id` required on TP | `validate:"required"` on `TrackingPlanV1.LocalID` | Tracking plan must have non-empty `id` |
| 3 | `description` constraints | `validate:"omitempty,gte=3,lte=2000,pattern=letter_start"` on `Description` | If present, 3-2000 chars starting with a letter |
| 4 | Rules dive | `validate:"omitempty,dive"` on `Rules` | Recursively validates each `TPRuleV1` |
| 5 | Rule `type` required + event_rule | `validate:"required,eq=event_rule"` on `TPRuleV1.Type` | Each rule type must be exactly `"event_rule"` |
| 6 | Rule `id` required | `validate:"required"` on `TPRuleV1.LocalID` | Each rule must have a non-empty `id` |
| 7 | Rule `event` required + format | `validate:"required,pattern=event_ref"` on `TPRuleV1.Event` | Event ref required (e.g. `#events:<id>`); implies properties without event are invalid |
| 8 | Rule `identity_section` | `validate:"omitempty,oneof=properties traits context.traits"` on `TPRuleV1.IdentitySection` | Same as V0 |
| 9 | Rule properties dive | `validate:"omitempty,dive"` on `TPRuleV1.Properties` | Recursively validates each `TPRulePropertyV1` |
| 10 | Property `property` required + format | `validate:"required,pattern=property_ref"` on `TPRulePropertyV1.Property` | Must match `#properties:<id>` |
| 11 | Nested properties dive | `validate:"omitempty,dive"` on `TPRulePropertyV1.Properties` | Recursively validates nested properties |

### Custom Logic (manual rule code)

Do **not** add custom logic for event vs includes (includes is deprecated). Event is required via tags above.

| # | Validation | Description |
|---|-----------|-------------|
| 12 | Nesting depth <= 3 levels | Nested properties within rules must not exceed 3 levels of nesting depth |
| 13 | `additionalProperties` only for nested properties | The `additionalProperties` field on `TPRulePropertyV1` is only valid for properties that have nested sub-properties |

---

## Semantic Validations to Add for V1

V1 semantic rule must use **`V1VersionPatterns(localcatalog.KindTrackingPlansV1)`**. Do not validate variants in the tracking plan semantic rule (variants are handled separately).

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Event refs resolve | The `event` string (format `#events:<id>`) must resolve to an existing event in the graph |
| 2 | Property refs resolve | Each `property` field in rule properties (format `#properties:<id>`) must resolve to an existing property in the graph |
| 3 | Nested property refs resolve and parent allows nesting | Nested property refs must resolve, and the parent property must be of type `object` or `array` with item type `object` |
| 4 | `display_name` uniqueness | No two tracking plans may share the same display_name |

### Reuse existing functions

When implementing V1 semantic validations, **reuse existing functions wherever possible**:

- **Ref resolution (#1–#2)**: Use **`funcs.ValidateReferences(spec, graph)`**. It walks any struct via reflection, finds string fields whose `validate` tag includes a pattern ending in `_ref` (e.g. `event_ref`, `property_ref`), and checks each ref in the graph. Pass `TrackingPlanV1` so event and property refs are validated without duplicate logic.
- **Display_name uniqueness (#4)**: Use **`trackingPlanDisplayNameCountMap(graph)`**. This function returns a `map[string]int` of display_name → count for all tracking plans in the graph. Both V0 and V1 semantic validators should call it, then check `countMap[spec.Name] > 1` and return the appropriate error (V1 can use a message that references kind `tracking-plan`). Do not duplicate the count-map generation.
- **Nested property refs and parent allows nesting (#3)**: Reuse the same approach as V0: share **`nestingAllowed(propertyType, config)`** and the same “resolve ref → check type supports nesting → recurse” logic. The V1 implementation should operate on `TPRulePropertyV1` (using `.Property` instead of `.Ref`) but call the same nesting-allowed and graph-lookup helpers so only the struct shape differs, not the rules.

Note: Rule `id` uniqueness within a TP and `id` uniqueness across TPs are handled by the project-level rule `project/duplicate-local-id` which is version-agnostic. Include refs and variant discriminator validation are not part of tracking plan validation (includes is deprecated; variants are validated elsewhere).

---

## Acceptance Criteria

- [ ] `validate:` tags added to `TrackingPlanV1`, `TPRuleV1`, `TPRulePropertyV1` structs (Event required + `event_ref`, Type `required,eq=event_rule`, identity_section same as V0; no variant tags)
- [ ] V0 `TPRule` has `Type` with `validate:"required,eq=event_rule"`
- [ ] V1 syntactic rule uses `V1VersionPatterns(localcatalog.KindTrackingPlansV1)` and `rules.ValidateStruct()` for tag-based validations (#1-#11)
- [ ] Custom logic only for nesting depth (#12) and additionalProperties constraint (#13); no event/includes handling
- [ ] No variant validation in tracking plan syntactic or semantic rules
- [ ] V1 semantic rule uses `V1VersionPatterns(localcatalog.KindTrackingPlansV1)` and implements the 4 semantic validations listed above
- [ ] Event refs use V1 format (string `#events:<id>` with `event_ref` tag)
- [ ] Property refs use V1 format (`property` field with `#properties:<id>`)
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

Add V1 spec validation rules for the `tracking-plan` resource in the datacatalog provider. These rules use `V1VersionPatterns(localcatalog.KindTrackingPlansV1)` to target `rudder/v1` specs with `kind: tracking-plan` (distinct from V0.1's `kind: tp`) and decode into `TrackingPlanV1`/`TPRuleV1` structs. No variant validation in tracking plan rules (handled separately). Event required via tags; no custom logic for event/includes.

---

## Changes

* Add `validate:` tags to `TrackingPlanV1`, `TPRuleV1`, `TPRulePropertyV1` (Event required + `event_ref`, Type `required,eq=event_rule`, identity_section same as V0; no variant tags). Add `Type` tag to V0 `TPRule`.
* Add V1 syntactic rule using `V1VersionPatterns` and `rules.ValidateStruct()` for tag-based validations + custom logic only for nesting depth and additionalProperties constraint
* Add V1 semantic rule using `V1VersionPatterns` for event/property reference resolution and display_name uniqueness (no include refs, no variant discriminator in TP rule)

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
