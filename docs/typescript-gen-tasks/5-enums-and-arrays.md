# Task 5: Enums & Arrays

**Depends on:** Task 1, 4

---

## Goal

Union types for enums, array types with item types.

---

## What to Build

### 1. Enum Types (Union Types)

Generate union types, NOT TypeScript enums:

```yaml
# Input
propConfig:
  enum: ["GET", "PUT", "POST"]
```

```typescript
// Output - CORRECT
type PropertyMethod = 'GET' | 'PUT' | 'POST';

// NOT this - WRONG
enum PropertyMethod { S_GET = 'GET', ... }
```

### 2. Integer Enums

```yaml
# Input
type: "integer"
propConfig:
  enum: [200, 201, 400, 500]
```

```typescript
// Output
type PropertyStatusCode = 200 | 201 | 400 | 500;
```

### 3. Array Types

```yaml
# Simple array
type: "array"
propConfig:
  itemTypes: ["string"]
```

```typescript
type PropertyTags = string[];
```

### 4. Multi-type Arrays

```yaml
# Multi-type array
propConfig:
  itemTypes: ["string", "integer"]
```

```typescript
type PropertyMixed = (string | number)[];
```

### 5. Object and Array Base Types

| YAML    | TypeScript            |
| ------- | --------------------- |
| `object`| `Record<string, any>` |
| `array` | `any[]` or `T[]`      |

---

## Acceptance Criteria

- [ ] Enums generate as union types (NOT TS enums)
- [ ] Integer enums generate as number literal unions
- [ ] Arrays generate with correct item types
- [ ] Multi-type arrays generate as `(T1 | T2)[]`
- [ ] Object type maps to `Record<string, any>`
- [ ] Array without itemTypes maps to `any[]`
