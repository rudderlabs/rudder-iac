# Task 6: Custom Types

**Depends on:** Task 1, 4, 5

---

## Goal

`CustomType{Name}` primitives and object types.

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

---

## Key Rules

1. **Naming:** Custom types use `CustomType` prefix
2. **Required/Optional:** Apply `?` modifier based on `required` field

---

## Acceptance Criteria

- [ ] Primitive custom types generate as `CustomType{Name}` aliases
- [ ] Object custom types generate as `CustomType{Name}` interfaces
- [ ] Properties referencing custom types resolve correctly
- [ ] JSDoc comments included from descriptions
