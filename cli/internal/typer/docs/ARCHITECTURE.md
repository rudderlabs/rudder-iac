# RudderTyper 2.0 Architecture

This document captures the core architectural decisions for RudderTyper 2.0 as defined in the [LLD](../../lld/RudderTyper%202%200%20232f2b415dd0806799c4c8398bdb5653.md). It serves as both context for LLMs and documentation for humans working on the codebase.

## Overview

RudderTyper 2.0 generates platform-specific RudderAnalytics bindings from tracking plans. The architecture emphasizes clear separation of concerns, extensibility, and maintainability.

![architecture diagram](./architecture.drawio.svg)

## Core Components

### Core Generator

- **Purpose**: Defines interfaces and common types for platform-specific code generation
- **Current Implementation**: Located in `cli/internal/typer/generator/core/`
- **Key Types**:

  ```go
  type File struct {
      Path    string  // Relative to output directory
      Content string  // File content as string
  }

  type GeneratorStrategy interface {
      Generate(plan plan.TrackingPlan, options GeneratorOptions) ([]File, error)
  }

  type GeneratorOptions struct {
      OutputDir string
      Platform  map[string]any
  }
  ```

### NameRegistry

- **Purpose**: Manages name collision resolution across all generated code constructs
- **Current Implementation**: Located in `cli/internal/typer/generator/core/name_registry.go`
- **Key Features**:
  - Registers names under scope/id pairs with bidirectional mapping
  - Applies configurable collision handlers
  - Returns existing names for duplicate registrations
  - Validates input parameters
- **Interface**:
  ```go
  type CollisionHandler func(name string, existingNames []string) string
  func (nr *NameRegistry) RegisterName(id string, scope string, name string) (string, error)
  func DefaultCollisionHandler(name string, existingNames []string) string
  ```

### Platform Generators

- **Purpose**: Platform-specific code generation implementations
- **Current Implementation**: Kotlin generator in `cli/internal/typer/generator/platforms/kotlin/`
- **Architecture**:
  - Direct function-based approach rather than full strategy pattern
  - Template-based code generation using Go embed
  - Context-driven template rendering
  - **Extraction Pattern**: Generators use TrackingPlan helper methods for data extraction instead of implementing custom traversal logic
- **Kotlin Generator**:
  ```go
  func Generate(plan *plan.TrackingPlan) ([]*core.File, error)
  ```
- **Data Flow**: Plan extraction → Context transformation → Template rendering

### Template System

- **Purpose**: Handles code generation using Go templates
- **Current Implementation**: Embedded templates in platform packages
- **Features**:
  - Go template engine with embed directives
  - Sub-template composition
  - Context-driven rendering
- **Example**: `typealias.tmpl` for Kotlin type aliases

### JSON Schema Plan Provider

- **Purpose**: Parses JSON Schema definitions into TrackingPlan domain models
- **Current Implementation**: Located in `cli/internal/typer/plan/providers/jsonschema.go`

## Core Models

### TrackingPlan Model

**Design Principle**: Direct representation of RudderStack domain entities as configured by users.

**What IS included**:

- ✅ Core domain entities: Event, Property, CustomType, TrackingPlan, EventRule
- ✅ Configuration types: PrimitiveType, PropertyConfig, Variant, VariantCase
- ✅ Domain-specific information relevant to code generation

**What is NOT included**:

- ❌ Derived information (e.g., JSON schema semantics)
- ❌ Generated code specific semantics (e.g., Nullable flags)
- ❌ Platform-specific constructs

**Benefits**:

- Clear reasoning about generation logic
- Input source independence (API, YAML, JSON, etc.)
- Domain-driven design alignment

**Current Implementation**: Located in `cli/internal/typer/plan/plan.go`

**Helper Methods**:

The TrackingPlan model provides convenience methods for extracting nested entities:

- `ExtractAllCustomTypes()` - Recursively extracts all custom types from event rules and their schemas
- `ExtractAllProperties()` - Recursively extracts all properties from event rules and nested schemas

These helper methods provide convenient access to nested data without violating domain model principles. They extract existing entities rather than deriving new information, maintaining the separation between domain representation and processing logic.

### TemplateContext Model

**Design Principle**: Direct representation of code constructs with minimal template logic.

**What IS included**:

- ✅ Structs describing code constructs (e.g., Kotlin data classes)
- ✅ Pre-processed information (escaped, formatted)
- ✅ Construct-specific information (comments, attributes)

**What is NOT included**:

- ❌ Raw tracking plan data
- ❌ Shared literals requiring template processing
- ❌ Business logic for code generation

**Template Design Principles**:

- **Platform-specific contexts**: Each platform defines context types for all supported code constructs (e.g., Kotlin supports both type aliases for primitive types and data classes for object types)
- **Semantic vs. syntax separation**: Context values contain semantic content (what to generate), while templates handle syntax-level formatting (escaping, quoting) via template functions
- **Pre-processed semantics**: All semantic decisions (names, types, structure) are made in generators and stored in context
- **Minimal conditional logic**: Templates focus on formatting rather than business logic
- **Construct-specific modeling**: Different context types for different code constructs ensure templates remain simple and focused

**Current Implementation**: Located in `cli/internal/typer/generator/platforms/kotlin/context.go`

## Design Principles

### Separation of Concerns

- TrackingPlan: Pure domain representation (platform-agnostic)
- TemplateContext: Platform-specific code construct representation with semantic content
- Generators: Business logic transformation layer (semantic decisions)
- Templates: Syntax-level formatting (escaping, indentation, quoting) via template functions

### Extensibility

- Strategy pattern for platform-specific generation
- Configurable collision handlers
- Pluggable plan providers
- Template-based code generation

### Maintainability

- Clear component boundaries
- Minimal dependencies between layers
- Domain-driven model design
- Comprehensive testing strategy

### Input Independence

- TrackingPlan model works with any input source
- No coupling to specific APIs or file formats
- Consistent generation logic across input types

## Testing Strategy

### Running Tests

Tests are executed using the Makefile at the project root:

```bash
make test
```

This runs all tests across the project, including the typer generator tests that validate the plan model functionality.

### Reference Tracking Plan

The testing approach leverages a comprehensive reference tracking plan that provides consistent test data across all components:

- **Location**: `cli/internal/typer/plan/testutils/reference_plan.go`
- **Purpose**: Provides known test data with predictable structure for reliable testing
- **Coverage**: Includes primitive custom types (email, age, active, null_type), object custom types (user_profile), nested properties, multi-type properties with sealed classes, variant support, null type support, and comprehensive event rules for all RudderStack event types
- **Event Types**: Covers Track events (with custom names), Identify, Page, Screen, and Group events with both Properties and Traits sections
- **Test Validation**: Tests dynamically validate that all properties and custom types in the reference maps are correctly extracted from the tracking plan

#### Modifying the Reference Plan

When adding new test data to the reference tracking plan, you **MUST** update multiple locations to maintain test consistency:

1. **Add to ReferenceProperties or ReferenceCustomTypes maps** in `reference_plan.go`
2. **Add to event rules** in `GetReferenceTrackingPlan()` function (if the property/type should be used in events)
   - Note: Properties and custom types are only extracted if they are actually used in event rules
   - Unused items in the reference maps will not be tested
3. **Regenerate platform testdata** using `make typer-kotlin-update-testdata` to update expected generated code
4. **Run tests** to verify the changes: `go test ./cli/internal/typer/...`

**Common Mistakes to Avoid**:
- ❌ Adding properties/custom types to reference maps but not using them in any event rules (they won't be tested)
- ❌ Forgetting to regenerate testdata after reference plan changes
- ❌ Not running tests after making changes

**Workflow Example**:
```go
// 1. Add property to reference plan
ReferenceProperties["new_field"] = &plan.Property{
    Name: "new_field",
    Description: "New test field",
    Types: []plan.PropertyType{plan.PrimitiveTypeString},
}

// 2. Use it in an event rule (otherwise it won't be extracted/tested)
rules = append(rules, plan.EventRule{
    Event: *ReferenceEvents["Some Event"],
    Schema: plan.ObjectSchema{
        Properties: map[string]plan.PropertySchema{
            "new_field": {
                Property: *ReferenceProperties["new_field"],
                Required: true,
            },
        },
    },
})

// 3. Regenerate testdata
// Run: make typer-kotlin-update-testdata

// 4. Run tests to verify
// Run: go test ./cli/internal/typer/...
```

### Benefits

- **Consistency**: All tests use the same reference data, ensuring consistent behavior across components
- **Reliability**: Known structure enables precise validation of extraction and generation logic
- **Maintainability**: Centralized test data reduces duplication and simplifies updates
- **Documentation**: Reference plan serves as living documentation of supported features

### Testing Approach

The plan model is tested indirectly through platform generator tests rather than direct unit tests. This approach:

- **Validates the complete pipeline**: Tests the entire flow from plan model through code generation
- **Reduces test maintenance**: Changes to the plan model are automatically tested through generator tests
- **Provides comprehensive coverage**: The reference plan includes diverse scenarios that exercise all plan model features

Platform generators (like the Kotlin generator) use the reference tracking plan to validate their output against expected generated code, ensuring both plan model correctness and generation logic accuracy for all plan scenarios.

### Docker-based Validation

The Kotlin generator includes an additional layer of validation through a Docker-based runtime verification system:

- **Location**: `cli/internal/typer/generator/platforms/kotlin/validator/`
- **Purpose**: Validates that generated Kotlin code not only matches expected output but also compiles and executes correctly in a real Kotlin runtime environment
- **Implementation**: Complete Docker-containerized Kotlin project with Gradle build system and RudderStack SDK integration
- **Test Coverage**: Comprehensive validation scenarios covering all supported RudderTyper features including:
  - All RudderStack event types (Track, Identify, Page, Screen, Group)
  - Custom types and properties with various data types
  - Enum handling and array support
  - Edge cases with minimal and comprehensive data sets

**Key Benefits**:

- **Runtime Verification**: Ensures generated code compiles and executes without errors
- **SDK Integration Testing**: Validates compatibility with actual RudderStack Kotlin SDK
- **Environment Consistency**: Docker containerization provides consistent testing environment across different development machines
- **End-to-End Validation**: Tests the complete pipeline from plan model to executable Kotlin code

**Usage**:

```bash
make typer-validate-kotlin
```

The validator uses the same test data as the unit tests (`testdata/Main.kt`), ensuring consistency between static code generation tests and runtime validation tests.

## Implementation Guidelines

### Platform-Specific Options

Generators can accept platform-specific options to customize code generation behavior. This pattern enables flexibility while maintaining a clean separation between generic and platform-specific configuration.

#### How Options Work

1. **Options Flow**: CLI `--option key=value` → Parsed map → RudderTyper orchestrator → Mapstructure decoding → Generator receives typed struct
2. **Automatic Validation**: Unknown options are rejected by the mapstructure decoder with `ErrorUnused: true`
3. **Defaults**: Each generator provides sensible defaults via `DefaultOptions()` method
4. **Struct-Based**: Options are defined as simple Go structs with struct tags for metadata

#### Implementing Platform Options

Each platform generator must implement the `core.Generator` interface:

```go
// Define platform-specific options struct with metadata tags
type KotlinOptions struct {
    PackageName string `mapstructure:"packageName" description:"Package name for generated Kotlin code (e.g., com.example.analytics)"`
}

// Define the Generator struct
type Generator struct{}

// Implement Generate method
func (g *Generator) Generate(plan *plan.TrackingPlan, options core.GenerateOptions, platformOptions any) ([]*core.File, error) {
    // platformOptions is already decoded to KotlinOptions by RudderTyper
    kotlinOpts := platformOptions.(KotlinOptions)

    // Validate options
    if err := kotlinOpts.Validate(); err != nil {
        return nil, err
    }

    // Use options in code generation
    // ...
}

// Provide default options (used for validation and CLI discovery)
func (g *Generator) DefaultOptions() any {
    return KotlinOptions{
        PackageName: "com.rudderstack.ruddertyper",
    }
}

// Validate options after decoding (called by generator, not orchestrator)
func (opts *KotlinOptions) Validate() error {
    if opts.PackageName != "" && !isValidPackageName(opts.PackageName) {
        return fmt.Errorf("invalid package name %q", opts.PackageName)
    }
    return nil
}
```

**Struct Tags**:
- `mapstructure:"key"` - Used by orchestrator to decode from `map[string]string` to struct fields
- `description:"..."` - Used by CLI to display help text

#### Registration

Add your generator to the platform registry in `cli/internal/typer/generator/platforms.go`:

```go
var platforms = map[string]core.Generator{
    "kotlin": &kotlin.Generator{},
    // Add new platforms here
}
```

#### CLI Usage

Users pass platform-specific options using repeatable `--option key=value` flags:

```bash
# Generate with custom Kotlin package
rudder typer generate --platform kotlin --tracking-plan-id ABC123 \
  --option packageName=com.example.analytics

# Discover available options for a platform
rudder typer options --platform kotlin
```

### Adding New Platforms

1. Create a new package under `cli/internal/typer/generator/platforms/{platform}/`
2. Define platform-specific context types (like `KotlinContext`)
3. Implement a `Generate` function following the current pattern
4. **Use plan helper methods** (`ExtractAllCustomTypes`, `ExtractAllProperties`) for data extraction instead of implementing custom traversal logic
5. **Process event rules** to generate platform-specific constructs for all RudderStack event types (Track, Identify, Page, Screen, Group) with appropriate naming conventions for your target platform
6. Create embedded Go templates using `//go:embed`
7. Implement platform-specific collision handlers and naming functions
8. Add template processing functions
9. **Leverage the reference tracking plan** for comprehensive testing of your generator

### Extending TrackingPlan Model

- Add new fields only if they represent domain concepts
- Maintain backwards compatibility
- Avoid derived or computed fields
- Document business justification

### Template Development

- Keep business logic minimal (no semantic decisions)
- Handle syntax-level formatting via template functions (escaping, quoting)
- Use specific context types for different constructs
- Avoid raw tracking plan access in templates
