# 02: Custom Types & Nested Objects

**Priority:** P1
**Depends on:** 01-foundation-and-types
**Complexity:** Medium

---

## Goal
Generate custom type definitions and handle nested object structures with inline types.

---

## What to Build

### 1. Primitive Custom Types
```yaml
# Input (custom-types.yaml)
- id: "some-string-type"
  name: "SomeStringType"
  type: "string"
  description: "some string custom type"
```
```typescript
// Output
/** some string custom type */
type CustomTypeSomeStringType = string;
```

### 2. Object Custom Types
```yaml
# Input (custom-types.yaml)
- id: "some-object-type"
  name: "SomeObjectType"
  type: "object"
  description: "some object custom type"
  properties:
    - $ref: "#/properties/typer-test/some-custom-string"
      required: true
    - $ref: "#/properties/typer-test/some-integer"
      required: false
```
```typescript
// Output
/** some object custom type */
interface CustomTypeSomeObjectType {
  someCustomString: PropertySomeCustomString;
  someInteger?: PropertySomeInteger;
}
```

### 3. Custom Type References
When a property references a custom type:
```yaml
# Input (properties.yaml)
- id: "some-custom-string"
  name: "someCustomString"
  type: "#/custom-types/typer-test/some-string-type"
```
```typescript
// Output
type PropertySomeCustomString = CustomTypeSomeStringType;
```

### 4. Nested Objects (INLINE Types)
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

1. **Naming:** Custom types use `CustomType` prefix
2. **Nesting:** Always inline, never separate interfaces
3. **Required/Optional:** Apply `?` modifier at each nesting level based on `required` field
4. **Depth:** Support arbitrary nesting depth (3+ levels)

---

## Acceptance Criteria
- [ ] Primitive custom types generate as `CustomType{Name}` aliases
- [ ] Object custom types generate as `CustomType{Name}` interfaces
- [ ] Properties referencing custom types resolve correctly
- [ ] Nested objects generate as inline types
- [ ] Deep nesting (3+ levels) works correctly
- [ ] Required/optional modifiers applied at all nesting levels
- [ ] JSDoc comments included from descriptions
