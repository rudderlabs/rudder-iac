# Task 7: Nesting

**Depends on:** Task 1, 4, 5, 6

---

## Goal

Inline nested objects in event properties.

---

## What to Build

### 1. Nested Objects (INLINE Types)

**Important:** Use inline types, NOT separate interfaces.

```yaml
# Input (tracking-plan.yaml)
properties:
  - $ref: "#/properties/typer-test/some-nested-object"
    properties:
      - $ref: "#/properties/typer-test/some-nested-level-1"
        properties:
          - $ref: "#/properties/typer-test/some-nested-level-2"
            properties:
              - $ref: "#/properties/typer-test/some-string"
              - $ref: "#/properties/typer-test/some-integer"
```

```typescript
// Output - CORRECT (inline)
interface TrackSomeTrackEventProperties {
  someNestedObject?: {
    someNestedLevel1?: {
      someNestedLevel2?: {
        someString?: PropertySomeString;
        someInteger?: PropertySomeInteger;
      };
    };
  };
}

// NOT this - WRONG (separate interfaces)
interface SomeNestedLevel2 { ... }
interface SomeNestedLevel1 { someNestedLevel2?: SomeNestedLevel2; }
```

---

## Key Rules

1. **Nesting:** Always inline, never separate interfaces
2. **Required/Optional:** Apply `?` modifier at each nesting level based on `required` field
3. **Depth:** Support arbitrary nesting depth (3+ levels)

---

## Acceptance Criteria

- [ ] Nested objects generate as inline types
- [ ] Deep nesting (3+ levels) works correctly
- [ ] Required/optional modifiers applied at all nesting levels
