# Kotlin Generator

This package generates type-safe Kotlin bindings for RudderStack tracking plans, enabling compile-time validation of analytics events.

## Overview

The Kotlin generator transforms tracking plan definitions into:

- **Type aliases** for primitive custom types and properties
- **Data classes** for object custom types and event properties/traits
- **Enum classes** for properties with enum constraints
- **Sealed classes** for variant types (discriminated unions) and multi-type properties/array items
- **Wrapper methods** in a `RudderAnalytics` class for type-safe event tracking

## Architecture

### Key Files

```
kotlin/
├── README.md                    # This file
├── generator.go                 # Main generation orchestrator
├── context.go                   # Platform-specific context types
├── naming.go                    # Identifier naming and sanitization
├── escape.go                    # String/comment escaping functions
├── templates.go                 # Template registration and rendering
├── rudderanalytics.go          # RudderAnalytics wrapper method generation
├── variants.go                  # Variant (sealed class) generation
├── templates/                   # Go templates for code generation
│   ├── Main.kt.tmpl            # Main file template
│   ├── typealias.tmpl          # Type alias template
│   ├── dataclass.tmpl          # Data class template
│   ├── enum.tmpl               # Enum class template
│   ├── sealedclass.tmpl        # Sealed class template
│   └── rudderanalytics.tmpl    # RudderAnalytics method template
├── testdata/
│   └── Main.kt                 # Expected generated output for tests
└── validator/                   # Docker-based runtime validation
```

### Design Principles

#### 1. Semantic vs. Syntax Separation

**Context structs contain semantic content (what to generate):**

- Unescaped strings (e.g., event name: `Product "Premium" Clicked`)
- Type information (e.g., `Type: "String"`)
- Structural information (e.g., `Nullable: true`)
- Values with explicit type flags (e.g., `IsLiteral: true`)

**Templates handle syntax-level formatting:**

- Escaping special characters via template functions (`escapeString`, `escapeComment`)
- Formatting literals via template functions (`formatLiteral`)
- Adding Kotlin syntax (quotes, annotations, etc.)

**Example:**

```go
// Context (semantic)
SDKCallArgument{
    Name: "name",
    Value: "Product \"Premium\" Clicked",  // Unescaped
    IsLiteral: true,                       // Flag: this is a literal
}

// Template (syntax) - rudderanalytics.tmpl
{{ .Name }} = {{ if .IsLiteral }}{{ formatLiteral .Value }}{{ else }}{{ .Value }}{{ end }}

// Output (Kotlin)
name = "Product \"Premium\" Clicked"  // Properly escaped
```

#### 2. Identifier Sanitization

All Kotlin identifiers (class names, method names, property names) must be valid according to Kotlin's identifier rules. The generator sanitizes invalid characters **before** converting to PascalCase/camelCase:

**Process:**

1. Input: `Product "Premium" Clicked` (from tracking plan)
2. Sanitize: `Product  Premium  Clicked` (quotes → spaces)
3. Format: `ProductPremiumClicked` (PascalCase)
4. Use in identifier: `class TrackProductPremiumClickedProperties`

**Sanitization rules** (`sanitizeForIdentifier`):

- Keep: Letters, digits, `_`, `-`, space, `.`
- Replace with space: All other characters (quotes, backslashes, etc.)
- Rationale: Spaces become word boundaries in PascalCase/camelCase conversion

**Applied to:**

- `FormatClassName` - for class/type alias names
- `formatPropertyName` - for property names
- `FormatMethodName` - for method names

#### 3. Escaping Special Characters

Three types of escaping are needed:

**a) String Literal Escaping** (`EscapeKotlinStringLiteral`):

```kotlin
// Input: Product "Premium" Clicked
// Output: Product \"Premium\" Clicked
name = "Product \"Premium\" Clicked"
```

**b) Comment Escaping** (`EscapeKotlinComment`):

```kotlin
// Input: User's email /* important */
// Output: User's email /\* important *\/
/** User's email /\* important *\/ */
```

**c) Literal Formatting** (`FormatKotlinLiteral`):
Handles different Go types and formats them as Kotlin literals:

- `string` → `"escaped string"`
- `int/int32/int64` → `42`
- `float32/float64` → `3.14` (trailing zeros removed via `%g`)
- `bool` → `true` / `false`
- `nil` → `` (empty)

### Context Types

#### Core Context Struct

```go
type KotlinContext struct {
    TypeAliases            []KotlinTypeAlias       // Primitive type aliases
    DataClasses            []KotlinDataClass       // Object types
    SealedClasses          []KotlinSealedClass     // Variant types
    Enums                  []KotlinEnum            // Enum constraints
    RudderAnalyticsMethods []RudderAnalyticsMethod // Wrapper methods
    EventContext           map[string]string       // Fixed event context
}
```

#### Method Argument Context

```go
type KotlinMethodArgument struct {
    Name             string // e.g., "userId", "properties"
    Type             string // e.g., "String", "TrackEventProperties"
    Nullable         bool   // Whether type is nullable (String?)
    Default          any    // Default value (nil = no default)
    IsLiteralDefault bool   // If true, format Default as literal
}
```

**Usage in templates:**

```kotlin
// Template
fun {{ .Name }}({{- range .MethodArguments -}}
    {{ .Name }}: {{ .Type }}{{ if .Nullable }}?{{ end }}
    {{- if .IsLiteralDefault }} = {{ formatLiteral .Default }}
    {{- else if .Default }} = {{ .Default }}{{ end }}
{{- end }}) { ... }

// Generated
fun identify(userId: String = "", traits: IdentifyTraits) { ... }
```

#### SDK Call Argument Context

```go
type SDKCallArgument struct {
    Name            string // Parameter name
    Value           any    // Value (string, int, bool, etc.)
    ShouldSerialize bool   // Whether to serialize to JsonObject
    IsLiteral       bool   // If true, format Value as literal
}
```

**Usage:**

```go
// Context
SDKCallArgument{
    Name: "name",
    Value: "Event Name",
    IsLiteral: true,
}

// Generated Kotlin
analytics.track(
    name = "Event Name",
    ...
)
```

## Type Mappings

### Primitive Type to Kotlin Type

The generator maps tracking plan primitive types to Kotlin types as follows:

| Primitive Type | Kotlin Type         | Description                     |
| -------------- | ------------------- | ------------------------------- |
| `string`       | `String`            | Text values                     |
| `integer`      | `Long`              | Whole numbers                   |
| `number`       | `Double`            | Decimal numbers                 |
| `boolean`      | `Boolean`           | true/false values               |
| `object`       | `JsonObject`        | Object without defined schema   |
| `array`        | `List<JsonElement>` | Array without defined item type |
| `null`         | `JsonNull`          | JSON null literal               |
| `any`          | `JsonElement`       | Any JSON value                  |

**Notes:**

- Arrays with defined item types use `List<ItemType>` (e.g., `List<String>`, `List<CustomTypeEmail>`)
- Multi-type properties/items generate sealed classes with subclasses for each type
- Properties with enum constraints generate enum classes instead of primitive types

### Multi-Type Support

Properties or array items with multiple types are represented as sealed classes:

```kotlin
// Property with types: ["string", "null"]
sealed class PropertyStringOrNull {
    abstract val _jsonElement: JsonElement

    data class StringValue(val value: String) : PropertyStringOrNull() {
        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }

    data class NullValue(val value: JsonNull) : PropertyStringOrNull() {
        override val _jsonElement: JsonElement = value
    }
}
```

Each type in the union gets a corresponding subclass:

- `StringValue` for `string`
- `IntegerValue` for `integer`
- `NumberValue` for `number`
- `BooleanValue` for `boolean`
- `ObjectValue` for `object`
- `ArrayValue` for `array`
- `NullValue` for `null`

## Testing

### Unit Tests

**Location:** `generator_test.go`

**What they test:**

- Complete generation pipeline from tracking plan to Kotlin code
- Generated output matches expected `testdata/Main.kt`
- Uses reference tracking plan from `cli/internal/typer/plan/testutils`

**Run tests:**

```bash
go test ./cli/internal/typer/generator/platforms/kotlin/...
```

### Regenerating Test Data

When you modify the generator or reference tracking plan, regenerate the expected output:

```bash
make typer-kotlin-update-testdata
```

**What this does:**

1. Runs `testutils/generate_reference_plan.go`
2. Generates Kotlin code from reference tracking plan
3. Writes output to `testdata/Main.kt`
4. This file becomes the expected output for `TestGenerate`

**When to regenerate:**

- After modifying generator logic
- After changing reference tracking plan in `cli/internal/typer/plan/testutils/reference_plan.go`
- After updating templates
- After adding new features that change output format

### Docker-Based Validation

**Location:** `validator/`

**Purpose:** Validates that generated code actually compiles and runs in a real Kotlin environment.

**Run validation:**

```bash
make typer-kotlin-validate
```

**What it does:**

1. Builds a Docker image with Kotlin/Gradle/RudderStack SDK
2. Copies generated `testdata/Main.kt` into a Kotlin project
3. Runs `gradle build` to compile the code
4. Runs test scenarios that exercise the generated code
5. Validates runtime behavior

**Benefits:**

- Ensures generated code is syntactically valid Kotlin
- Tests SDK integration
- Catches issues that static tests might miss
- Provides consistent environment across machines

**Structure:**

```
validator/
├── Dockerfile              # Kotlin/Gradle environment
├── Makefile               # Build and run commands
├── build.gradle.kts       # Gradle build config
└── src/
    └── main/kotlin/
        ├── Main.kt        # Generated code (copied from testdata)
        └── TestMain.kt    # Test scenarios
```

## Common Tasks

### Adding a New Code Construct

**Example: Adding support for type aliases with custom annotations**

1. **Update context** (`context.go`):

```go
type KotlinTypeAlias struct {
    Alias       string
    Comment     string
    Type        string
    Annotations []string  // New field
}
```

2. **Update generator** (`generator.go`):

```go
alias := &KotlinTypeAlias{
    Alias:       finalName,
    Comment:     customType.Description,
    Type:        kotlinType,
    Annotations: []string{"@JvmInline"},  // Add annotations
}
```

3. **Update template** (`templates/typealias.tmpl`):

```kotlin
{{- if .Comment }}
/** {{ escapeComment .Comment }} */
{{- end }}
{{- range .Annotations }}
{{ . }}
{{- end }}
typealias {{ .Alias }} = {{ .Type }}
```

4. **Regenerate testdata** and verify:

```bash
make typer-kotlin-update-testdata
go test ./cli/internal/typer/generator/platforms/kotlin/...
```

### Adding Escaping for New Context

If you add a new field that contains user-provided text:

1. **Determine the context:**

   - String literal? Use `escapeString` in template
   - Comment? Use `escapeComment` in template
   - Already a literal value? Use `formatLiteral` in template

2. **Update template:**

```kotlin
// For string literals
@SerialName("{{ escapeString .NewField }}")

// For comments
/** {{ escapeComment .NewField }} */

// For literal values (with IsLiteral flag)
{{ if .IsLiteral }}{{ formatLiteral .Value }}{{ else }}{{ .Value }}{{ end }}
```

3. **Add test case** to reference plan with special characters

4. **Verify** in generated output

### Debugging Generated Code

**1. Check intermediate context:**
Add debug logging in `generator.go`:

```go
import "encoding/json"

// After building context
contextJSON, _ := json.MarshalIndent(ctx, "", "  ")
fmt.Println(string(contextJSON))
```

**2. Check template rendering:**

- Templates are in `templates/*.tmpl`
- Use `{{ .FieldName }}` to output values
- Use `{{- }}` to trim whitespace
- Use `{{ escapeComment .Comment }}` to apply functions

**3. Check naming:**
Add logging in `naming.go`:

```go
func FormatClassName(prefix, name string) string {
    fmt.Printf("FormatClassName: input=%q, sanitized=%q\n", name, sanitized)
    // ...
}
```

**4. Validate generated Kotlin:**

```bash
# Quick syntax check
make typer-kotlin-update-testdata
make typer-kotlin-validate
```

## Naming Conventions

### Generated Names

**Type Aliases (Primitive Custom Types):**

- Pattern: `CustomType{Name}`
- Example: `CustomTypeEmail`, `CustomTypeUserId`

**Data Classes (Object Custom Types):**

- Pattern: `CustomType{Name}`
- Example: `CustomTypeUserProfile`

**Data Classes (Event Properties/Traits):**

- Pattern: `{EventType}{EventName}{Section}`
- Examples:
  - `TrackUserSignedUpProperties`
  - `IdentifyTraits`
  - `ScreenProperties`

**Enum Classes:**

- Pattern: `Property{Name}` or `CustomType{Name}`
- Example: `PropertyDeviceType`, `CustomTypeStatus`

**Sealed Classes (Variants):**

- Pattern: `CustomType{Name}` or `Property{Name}`
- Subclasses: `Case{MatchValue}`
- Example: `CustomTypePageType.CaseSearch`, `CustomTypePageType.CaseProduct`

**Methods:**

- Pattern: `{verb}{EventName}` (camelCase)
- Examples: `trackUserSignedUp()`, `identify()`, `screen()`

**Properties:**

- Pattern: `{name}` (camelCase)
- Example: `firstName`, `emailAddress`, `deviceType`

### Collision Handling

The `NameRegistry` tracks all generated names and handles collisions by appending numbers:

- First: `UserProfile`
- Collision: `UserProfile2`
- Collision: `UserProfile3`

## Known Limitations

### 1. Enum Serialization

**Current behavior:** All enum values serialize as strings, regardless of original type.

```kotlin
enum class PropertyStatusCode {
    @SerialName("200")  // Serializes as "200" (string)
    _200,
    @SerialName("404")  // Serializes as "404" (string)
    _404
}
```

**Impact:** Numeric enum values become strings in JSON.

**Future work:** Custom serializers for type-preserving enum serialization (tracked separately).

### 2. Identifier Sanitization Trade-offs

**Sanitization removes semantic information:**

```
Input:  "Product \"Premium\" Clicked"
Output: "ProductPremiumClicked"
```

The quotes are lost in the identifier but preserved in the string literal used at runtime.

**Alternative considered:** Encoding special chars (e.g., `Product_Q_Premium_Q_Clicked`) - rejected for readability.

### 3. Template Function Limitations

Go templates have limited logic:

- No variable assignments
- No complex conditionals
- No loops with state

**Solution:** Pre-process all logic in generator, keep templates simple.

## Best Practices

### 1. Keep Templates Simple

**Good:**

```kotlin
{{ if .Comment }}/** {{ escapeComment .Comment }} */{{ end }}
@SerialName("{{ escapeString .SerialName }}")
```

**Bad:**

```kotlin
{{ if and .Comment (ne .Comment "") }}
    {{ $escaped := escapeComment .Comment }}
    /** {{ $escaped }} */  <!-- Variable assignment not supported! -->
{{ end }}
```

### 2. Pre-process in Generator

Move logic to generator, store results in context:

```go
// Generator
prop := &KotlinProperty{
    Name:       formatPropertyName(propName),
    SerialName: propName,
    Type:       kotlinType,
    Nullable:   !required,
    Comment:    description,  // Store unescaped
}

// Template
{{ if .Comment }}/** {{ escapeComment .Comment }} */{{ end }}
```

### 3. Test Edge Cases

Always test with special characters:

- Event names with quotes: `Product "Premium" Clicked`
- Comments with `*/`: `Email /* important */`
- Enum values with spaces/symbols: `200: OK`
- Unicode: `世界`
- Empty strings
- Very long names

### 4. Validate with Docker

Don't rely solely on unit tests - always validate generated code compiles:

```bash
make typer-kotlin-validate
```

## Platform-Specific Options

The Kotlin generator supports options to customize code generation behavior. Options are defined using struct tags and validated after being decoded from CLI arguments.

### Available Options

#### `packageName`

**Type:** `string`
**Default:** `com.rudderstack.ruddertyper`
**Description:** Sets the package name for generated Kotlin code

##### Validation Rules

- Must be lowercase (regex: `^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)
- Can contain letters, digits, and underscores
- Segments separated by dots
- Each segment must start with a letter
- Cannot start or end with a dot
- Cannot have consecutive dots

## References

- [Main Architecture Doc](../../docs/ARCHITECTURE.md)
- [Kotlin Language Spec](https://kotlinlang.org/spec/)
- [RudderStack Kotlin SDK](https://github.com/rudderlabs/rudder-sdk-kotlin)
