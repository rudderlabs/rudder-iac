# Task 4: Testing

**Depends on:** Task 1, 2, 3

---

## Goal

Golden file snapshot test + `tsc --noEmit` compilation check (like Kotlin generator).

---

## What to Build

### 1. Unit Tests

Test individual components:

```go
// naming_test.go
func TestToCamelCase(t *testing.T) {
    tests := []struct{ input, expected string }{
        {"Some Track Event", "someTrackEvent"},
        {"user_signed_up", "userSignedUp"},
        {"$Variable$String", "variableString"},
    }
    // ...
}

func TestToPascalCase(t *testing.T) { ... }
func TestSanitizeName(t *testing.T) { ... }
func TestReservedWordHandling(t *testing.T) { ... }
```

```go
// types_test.go
func TestPrimitiveTypeMapping(t *testing.T) {
    tests := []struct{ yaml, ts string }{
        {"string", "string"},
        {"integer", "number"},
        {"boolean", "boolean"},
        {"null", "null"},
    }
    // ...
}

func TestEnumGeneration(t *testing.T) { ... }
func TestArrayTypeGeneration(t *testing.T) { ... }
func TestMultiTypeGeneration(t *testing.T) { ... }
```

### 2. Integration Tests

Test full generation against test data:

```go
// generator_test.go
func TestGenerateFromTestData(t *testing.T) {
    plan, err := plan.LoadFromDirectory("../../plan/testdata/project/")
    require.NoError(t, err)

    generator := &Generator{}
    files, err := generator.Generate(plan, options, nil)
    require.NoError(t, err)

    // Verify output structure
    require.Len(t, files, 1)
    require.Equal(t, "index.ts", files[0].Path)

    // Verify content contains expected elements
    content := files[0].Content
    assert.Contains(t, content, "export class RudderTyperAnalytics")
    assert.Contains(t, content, "type Property")
    assert.Contains(t, content, "interface Track")
}
```

### 3. Golden File Tests

Compare output against expected reference files:

```go
func TestGoldenOutput(t *testing.T) {
    // Generate output
    output := generateFromTestData()

    // Compare to golden file
    golden := readFile("testdata/golden/index.ts")

    if *updateGolden {
        writeFile("testdata/golden/index.ts", output)
    }

    assert.Equal(t, golden, output)
}
```

### 4. TypeScript Compilation Test

Verify generated code compiles:

```go
func TestTypeScriptCompiles(t *testing.T) {
    // Generate output
    output := generateFromTestData()

    // Write to temp file
    tmpDir := t.TempDir()
    indexPath := filepath.Join(tmpDir, "index.ts")
    os.WriteFile(indexPath, []byte(output), 0644)

    // Run tsc --noEmit
    cmd := exec.Command("npx", "tsc", "--noEmit", "--strict", indexPath)
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "TypeScript compilation failed: %s", output)
}
```

### 5. SDK Compatibility Test

Verify types work with actual SDK:

```typescript
// test/sdk-compat.ts
import type { RudderAnalytics } from "@rudderstack/analytics-js";
import { RudderTyperAnalytics, TrackSomeEventProperties } from "../index";

// This file should compile without errors

declare const analytics: RudderAnalytics;
const rudderTyper = new RudderTyperAnalytics(analytics);

// Test track
rudderTyper.trackSomeEvent({ someString: "hello" });
rudderTyper.trackSomeEvent({ someString: "hello" }, () => {});
rudderTyper.trackSomeEvent({ someString: "hello" }, {}, () => {});

// Test identify
rudderTyper.identify("user-123", { email: "test@test.com" });

// Test type safety (these should cause compile errors if uncommented)
// rudderTyper.trackSomeEvent({ wrongProp: 'hello' }); // Error!
// rudderTyper.trackSomeEvent('wrong type'); // Error!
```

---

## Test Commands

```bash
# Run unit tests
cd cli && go test ./internal/typer/generator/platforms/typescript/...

# Run all typer tests
cd cli && go test ./internal/typer/...

# Generate and compile check
go run ./cmd/rudder-cli typer generate --platform typescript --output ./test-output
cd test-output && npx tsc --noEmit --strict index.ts

# Full validation
cd test-output && npm install @rudderstack/analytics-js
npx tsc --noEmit --strict index.ts
```

---

## Test Data

### Required Test Events (add to events.yaml if missing)

Ensure test data includes ALL event types:

```yaml
# events.yaml - should have:
- id: "some_track_event"
  name: "Some Track Event"
  event_type: "track"

- id: "user_identify"
  name: "User Identify"
  event_type: "identify"

- id: "product_page_viewed"
  name: "Product Page Viewed"
  event_type: "page"

- id: "company_group"
  name: "Company Group"
  event_type: "group"

- id: "user_alias"
  name: "User Alias"
  event_type: "alias"
```

### Required Event Rules (add to tracking-plan.yaml if missing)

```yaml
# tracking-plan.yaml - should have rules for each event type:
- type: "event_rule"
  event:
    $ref: "#/events/typer-test/user_identify"
  properties:
    - $ref: "#/properties/typer-test/email"
      required: true

- type: "event_rule"
  event:
    $ref: "#/events/typer-test/product_page_viewed"
  properties:
    - $ref: "#/properties/typer-test/product-id"
      required: true

- type: "event_rule"
  event:
    $ref: "#/events/typer-test/company_group"
  properties:
    - $ref: "#/properties/typer-test/company-name"
      required: true

- type: "event_rule"
  event:
    $ref: "#/events/typer-test/user_alias"
```

### Test Data Location

```
cli/internal/typer/plan/testdata/project/
├── tracking-plan.yaml    # Event rules for ALL event types
├── events.yaml           # Events with track, identify, page, group, alias
├── properties.yaml
└── custom-types.yaml

cli/internal/typer/generator/platforms/typescript/testdata/
├── golden/
│   └── index.ts          # Expected output
└── sdk-compat/
    └── test.ts           # SDK compatibility test
```

---

## Acceptance Criteria

- [ ] Test data includes all event types (track, identify, page, group, alias)
- [ ] Unit tests for naming utilities (camelCase, PascalCase, sanitize, reserved words)
- [ ] Unit tests for type mapping (primitives, enums, arrays, multi-type)
- [ ] Integration test generates from test data without errors
- [ ] Golden file test compares output to expected
- [ ] Generated TypeScript compiles with `tsc --noEmit --strict`
- [ ] Generated types compatible with `@rudderstack/analytics-js`
- [ ] All existing Kotlin tests still pass
- [ ] CI pipeline includes TypeScript generation tests
