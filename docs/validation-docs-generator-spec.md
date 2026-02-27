# Validation Rules Documentation Generator Specification

This document specifies the implementation of the `rudder-cli validate --gen-docs` command that auto-generates validation rule documentation from the codebase.

## Goals

1. Generate comprehensive validation documentation from code
2. Support versioned documentation (latest in main doc, older versions in subdirectory)
3. Provide consistent documentation structure across all rules
4. Enable programmatic extraction of rule metadata via Registry interface

---

## Implementation Summary

### CLI Command

The documentation generator is integrated into the `validate` command with the `--gen-docs` flag:

```bash
# Generate documentation to stdout
rudder-cli validate --gen-docs

# Generate documentation to a specific file
rudder-cli validate --gen-docs --output ./docs/validation-rules.md
```

### Module Location

The generator is implemented in `cli/internal/ui/docgen.go`:

```go
package ui

import (
    "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// RuleDocGenerator generates markdown documentation for validation rules.
type RuleDocGenerator struct {
    registry rules.Registry
}

// NewRuleDocGenerator creates a new documentation generator.
func NewRuleDocGenerator(reg rules.Registry) *RuleDocGenerator

// Generate produces the complete markdown documentation for all registered rules.
func (g *RuleDocGenerator) Generate() string
```

### Registry Interface Extensions

The `rules.Registry` interface was extended to support documentation generation:

```go
type Registry interface {
    // ... existing methods ...

    // AllKinds returns all registered kinds (excluding wildcard "*").
    // This is useful for documentation generation.
    AllKinds() []string

    // AllRules returns all registered rules (both syntactic and semantic).
    // Each rule is returned only once, even if it applies to multiple kinds.
    // This is useful for documentation generation.
    AllRules() []Rule
}
```

---

## Document Structure

### Output Files

```
docs/
├── validation-rules.md           # Latest version (rudder/v1)
└── validation-rules/
    └── v0.1.md                   # Legacy version (rudder/0.1)
```

### Document Hierarchy

```markdown
# Validation Rules

> **Spec Version:** rudder/v1
> **Other Versions:** [v0.1](./validation-rules/v0.1.md)

## Table of Contents
- [Global](#global)
- [Properties](#properties)
- [Events](#events)
- [Categories](#categories)
- [Custom Types](#custom-types)
- [Tracking Plans](#tracking-plans)
- [Event Stream](#event-stream)
- [RETL](#retl)

## Global
[Global rules that apply to all spec kinds]

## Properties
[Properties-specific rules]

...
```

---

## Rule Template Structure

Each rule is documented using the following structure:

### Standard Rule Template

```markdown
### {provider}/{kind}/{rule-name}

![Severity: {severity}](https://img.shields.io/badge/severity-{severity}-{color})

**Description:** {description}

**Applies to:** `{kinds}`

**Validation Phase:** {syntactic|semantic}

**Available in:** [v1](./#anchor) | [v0.1](./validation-rules/v0.1.md#anchor)

**Checks Performed:**

{list of specific validations performed}

**Valid Examples:**

```yaml
{valid example}
```

**Invalid Examples:**

```yaml
{invalid example}
```

---
```

### Template Variables

| Variable | Description | Source |
|----------|-------------|--------|
| `{provider}` | Provider name (project, datacatalog, event-stream, retl) | Rule ID prefix |
| `{kind}` | Resource kind (properties, events, etc.) | Rule ID segment |
| `{rule-name}` | Rule name | Rule ID suffix |
| `{severity}` | error, warning, info | `Rule.Severity()` |
| `{color}` | red, yellow, blue | Derived from severity |
| `{description}` | Human-readable description | `Rule.Description()` |
| `{kinds}` | Comma-separated kinds or `*` | `Rule.AppliesTo()` |

### Severity Badge Colors

| Severity | Badge Color |
|----------|-------------|
| Error | red |
| Warning | yellow |
| Info | blue |

---

## Rule Interface

Each rule implements the following interface for documentation extraction:

```go
// Rule interface
type Rule interface {
    ID() string
    Severity() Severity
    Description() string
    AppliesTo() []string
    Examples() Examples
    Validate(ctx *ValidationContext) []ValidationResult
}

// Examples holds valid and invalid YAML examples for documentation
type Examples struct {
    Valid   []string
    Invalid []string
}

// HasExamples returns true if either Valid or Invalid examples exist
func (e Examples) HasExamples() bool
```

---

## Current Validation Rules Inventory

### Global Rules (project/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `project/metadata-syntax-valid` | metadata syntax must be valid | Syntactic |
| `project/spec-syntax-valid` | spec syntax must be valid | Syntactic |
| `project/spec-values-valid` | spec kind and version must be valid and supported | Syntactic |
| `project/duplicate-local-id` | local IDs must be unique within each kind | Project |

### Properties Rules (datacatalog/properties/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `datacatalog/properties/spec-syntax-valid` | property spec syntax must be valid | Syntactic |
| `datacatalog/properties/config-valid` | property config must be valid for the given type | Syntactic |
| `datacatalog/properties/semantic-valid` | property references must resolve to existing resources | Semantic |

### Events Rules (datacatalog/events/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `datacatalog/events/spec-syntax-valid` | event spec syntax must be valid | Syntactic |

### Categories Rules (datacatalog/categories/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `datacatalog/categories/spec-syntax-valid` | category spec syntax must be valid | Syntactic |
| `datacatalog/categories/semantic-valid` | category names must be unique across the catalog | Semantic |

### Custom Types Rules (datacatalog/custom-types/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `datacatalog/custom-types/spec-syntax-valid` | custom type spec syntax must be valid | Syntactic |
| `datacatalog/custom-types/config-valid` | custom type config must be valid for the given type | Syntactic |
| `datacatalog/custom-types/semantic-valid` | custom type references must resolve to existing resources | Semantic |

### Tracking Plans Rules (datacatalog/tracking-plans/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `datacatalog/tracking-plans/spec-syntax-valid` | tracking plan spec syntax must be valid | Syntactic |
| `datacatalog/tracking-plans/semantic-valid` | tracking plan references must resolve to existing resources | Semantic |

### Event Stream Rules (event-stream/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `event-stream/source/spec-syntax-valid` | event stream source spec syntax must be valid | Syntactic |
| `event-stream/source/semantic-valid` | event stream source references must resolve to existing resources | Semantic |

### RETL Rules (retl/*)

| Rule ID | Description | Phase |
|---------|-------------|-------|
| `retl/sqlmodel/spec-syntax-valid` | retl sql model spec syntax must be valid | Syntactic |

---

## Section Links for Doc Site Integration

Each rule section can be linked directly from resource-level reference documentation:

### Global Rules
- Main: `docs/validation-rules.md#global`
- v0.1: `docs/validation-rules/v0.1.md#global`

### Properties
- Main: `docs/validation-rules.md#properties`
- v0.1: `docs/validation-rules/v0.1.md#properties`

### Events
- Main: `docs/validation-rules.md#events`
- v0.1: `docs/validation-rules/v0.1.md#events`

### Categories
- Main: `docs/validation-rules.md#categories`
- v0.1: `docs/validation-rules/v0.1.md#categories`

### Custom Types
- Main: `docs/validation-rules.md#custom-types`
- v0.1: `docs/validation-rules/v0.1.md#custom-types`

### Tracking Plans
- Main: `docs/validation-rules.md#tracking-plans`
- v0.1: `docs/validation-rules/v0.1.md#tracking-plans`

### Event Stream
- Main: `docs/validation-rules.md#event-stream`
- v0.1: `docs/validation-rules/v0.1.md#event-stream`

### RETL
- Main: `docs/validation-rules.md#retl`
- v0.1: `docs/validation-rules/v0.1.md#retl`

---

## Version Mapping

| Spec Version | Document |
|--------------|----------|
| `rudder/v1` | `docs/validation-rules.md` (main) |
| `rudder/0.1`, `rudder/v0.1` | `docs/validation-rules/v0.1.md` |

---

## Appendix: Reference Patterns

### Legacy Reference Format
```
#/<kind>/<group>/<id>
```

Examples:
- `#/properties/user-props/user_id`
- `#/events/core-events/page_viewed`
- `#/custom-types/address-types/Address`
- `#/categories/event-cats/User Actions`
- `#/tp/tracking-plans/Main TP`

### New Reference Format (v1)
```
#<kind>:<id>
```

Examples:
- `#property:user_id`
- `#event:page_viewed`
- `#custom-type:Address`
- `#category:User Actions`
- `#tracking-plan:Main TP`

---

## Appendix: Primitive Types and Config Keywords

### Primitive Types
- `string`
- `number`
- `integer`
- `boolean`
- `array`
- `object`
- `null`

### Config Keywords by Type

| Type | Allowed Keywords |
|------|------------------|
| `string` | `enum`, `minLength`, `maxLength`, `pattern`, `format` |
| `integer` | `enum`, `minimum`, `maximum` |
| `number` | `enum`, `minimum`, `maximum` |
| `array` | `itemTypes`, `minItems`, `maxItems` |
| `boolean` | `enum` |
| `object` | (no config allowed) |
| `null` | (no config allowed) |

### Valid Format Values
- `date-time`
- `date`
- `time`
- `email`
- `uuid`
- `hostname`
- `ipv4`
- `ipv6`

### Valid Event Types
- `track`
- `screen`
- `identify`
- `group`
- `page`
