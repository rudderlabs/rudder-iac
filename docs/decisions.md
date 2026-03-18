# Design Decisions

## PRO-5407: Migrate Datagraph Inline Validation to Rule-Based Framework

### Match Patterns: V1-only

Data graph is a new provider with no legacy specs. The `rudder/v0.1` usage in tests
was copied from older providers but does not reflect real usage. All data-graph specs
use `rudder/v1`, and exports always produce `rudder/v1`.

**Decision**: Use V1-only match patterns (`V1VersionPatterns("data-graph")`) in all rules.

### Table Format Validation: Inline in Rule Function (not registered pattern)

The `table` field requires a domain-specific 3-part format (`catalog.schema.table`).
The plan suggested validating this inline rather than registering a global pattern.

**Decision**: Keep the regex inline in the syntactic rule function. This is a
data-graph-specific concern and doesn't warrant global pattern registration.

### Conditional Required Fields: Custom Validation (not struct tags)

Fields like `primary_id` (required when type=entity) and `timestamp` (required when
type=event) are conditionally required. The `required_if` tag is not supported by
`ParseValidationErrors`.

**Decision**: Handle conditional required fields as custom validation logic in the
syntactic rule function, after struct tag validation.

### Semantic Rules: Spec-Level Model Lookup

Semantic rules receive the `DataGraphSpec` (not individual resources). For cardinality
validation, the target model type needs to be resolved. Models may exist in the same
spec or in the graph from other specs.

**Decision**: Build a local model type map from the spec, then fall back to graph
lookup via `RawData()` type assertion. This handles both same-spec and cross-spec
references.

### Handler ValidateResource: Return nil (not removed)

`ValidateResource` is a required method on the `HandlerImpl` interface with no default
no-op implementation.

**Decision**: Keep the method but return nil in all three handlers (datagraph, model,
relationship). All validation is now handled by the rule framework.
