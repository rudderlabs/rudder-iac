# RudderTyper 2.0 Implementation Plan

## Overview

This implementation plan provides a phased approach for building RudderTyper 2.0, a code generation system that transforms tracking plans into platform-specific RudderAnalytics bindings. Each phase introduces incremental functionality and includes working tests to validate the implementation.

## Phase 1: Basic Infrastructure + Type Aliases - COMPLETED ✅

**Goal**: Generate Kotlin type aliases for primitive custom types

### Core Infrastructure

- [x] **Plan Models** (`cli/internal/typer/plan/`):
  - [x] Already exists with basic structure
- [x] **Name Registry** (`cli/internal/typer/generator/core/`):
  - [x] Create `name_registry.go` with basic implementation
  - [x] Define `CollisionHandler` function type: `func(name string, existingNames []string) string`
  - [x] Implement basic `NameRegistry` struct with `RegisterName()` method with error handling
  - [x] Create comprehensive tests for NameRegistry
- [x] **Generator Core** (`cli/internal/typer/generator/core/`):
  - [x] Create formatting utilities: `ToPascalCase`, `ToCamelCase`, `SplitIntoWords`
  - [x] Create comprehensive tests for formatting utilities
- [x] **Generator Core** (`cli/internal/typer/generator/core/`):
  - [x] Define basic `GeneratorStrategy` interface
  - [x] Define `File` struct for generated files
  - [x] Define `GeneratorOptions` for configuration

### Kotlin Type Alias Generation

- [x] **Context Types** (`cli/internal/typer/generator/kotlin/context.go`):
  - [x] Define `KotlinTypeAlias` struct with Alias, Comment, Type fields
  - [x] Define basic `KotlinContext` with TypeAliases slice
  - [x] Create `NewKotlinContext()` constructor
- [x] **Name Handling** (`cli/internal/typer/generator/kotlin/naming.go`):
  - [x] Implement `FormatClassName()` - PascalCase for type aliases with reserved keyword handling
  - [x] Kotlin-specific collision handler using NameRegistry
  - [x] Comprehensive tests for naming functions
- [x] **Generator Logic** (`cli/internal/typer/generator/kotlin/generator.go`):
  - [x] Implement `Generate()` method focusing only on primitive custom types
  - [x] Process primitive custom types → generate KotlinTypeAlias
  - [x] Register names with NameRegistry
  - [x] Alphabetical sorting for deterministic output
- [x] **Templates** (`cli/internal/typer/generator/kotlin/templates/`):
  - [x] Create basic `main.kt.tmpl` with package declaration and typealias section
  - [x] Implement `typealias.tmpl` partial
  - [x] Update template engine to support sub-templates

### Phase 1 Testing

- [x] **Test** (`cli/internal/typer/generator/kotlin/generator_test.go`):
  - [x] Update `TestGenerate()` to include basic type alias generation
  - [x] Build tracking plan with multiple primitive custom types (email→String, age→Int, etc.)
  - [x] Generate Kotlin code and validate type aliases are created correctly
  - [x] Verify generated code based on testdata output (`cli/internal/typer/generator/kotlin/testdata/Main.kt`)

**Deliverable**: ✅ Generate valid Kotlin file with type aliases for primitive custom types

---

## Phase 2: Object Custom Types (Data Classes)

**Goal**: Generate Kotlin data classes for object custom types

### Enhanced Custom Type Processing

- [ ] **Context Types** (`cli/internal/typer/generator/kotlin/context.go`):
  - [ ] Define `KotlinDataClass` struct with ClassName, Comment, Properties, IsSerializable
  - [ ] Define `KotlinProperty` struct with FieldName, SerialName, Type, Comment, IsOptional, DefaultValue
  - [ ] Add DataClasses slice to `KotlinContext`
- [ ] **Generator Logic** (`cli/internal/typer/generator/kotlin/generator.go`):
  - [ ] Extend `processCustomTypes()` to handle object custom types
  - [ ] Generate KotlinDataClass for object types
  - [ ] Process properties and create KotlinProperty structs
  - [ ] Handle nested object schemas (recursively)
- [ ] **Name Handling** (`cli/internal/typer/generator/kotlin/naming.go`):
  - [ ] Add `FormatPropertyName()` - camelCase for properties
- [ ] **Templates**:
  - [ ] Create `dataclass.tmpl` partial
  - [ ] Update `main.kt.tmpl` to include data classes section

### Phase 2 Testing

- [ ] **Test** (`cli/internal/typer/generator/kotlin/generator_test.go`):
  - [ ] Update `TestGenerate()` to include object custom types
  - [ ] Build tracking plan with both primitive and object custom types
  - [ ] Include nested object properties
  - [ ] Update testdata to validate generated data classes

**Deliverable**: Generate Kotlin file with type aliases AND data classes for object custom types

---

## Phase 3: Event Type Generation

**Goal**: Generate data classes for event properties and traits

### Event Processing

- [ ] **Generator Logic** (`cli/internal/typer/generator/kotlin/generator.go`):
  - [ ] Implement `processEventRules()` function
  - [ ] Generate data classes for event properties/traits
  - [ ] Use naming pattern: `{EventName}Properties`, `{EventName}Traits`
  - [ ] Handle different event sections (properties, traits, context.traits)
  - [ ] Reference existing custom types in event properties
- [ ] **Property Processing**:
  - [ ] Implement `processProperty()` function
  - [ ] Map primitive types to Kotlin types (string→String, number→Double, etc.)
  - [ ] Handle custom type references
  - [ ] Mark properties as required vs optional

### Phase 3 Testing

- [ ] **Test** (`cli/internal/typer/generator/kotlin/generator_test.go`):
  - [ ] Update `TestGenerate()` to include event properties
  - [ ] Build tracking plan with custom types AND events
  - [ ] Include events that reference custom types
  - [ ] Include different event types (track, identify, page)
  - [ ] Update testdata to validate generated data classes
  - [ ] Validate event-specific data classes are generated
  - [ ] Verify proper type references between events and custom types

**Deliverable**: Generate complete type system including custom types and event-specific types

---

## Phase 4: RudderAnalytics Methods

**Goal**: Generate type-safe analytics methods for each event

### Analytics Generation

- [ ] **Context Types** (`cli/internal/typer/generator/kotlin/context.go`):
  - [ ] Define `KotlinMethod` struct with MethodName, EventName, EventType, PropertiesType, Comment
  - [ ] Add RudderAnalyticsMethods slice to `KotlinContext`
- [ ] **Generator Logic** (`cli/internal/typer/generator/kotlin/generator.go`):
  - [ ] Implement `generateRudderAnalytics()` function
  - [ ] Create method for each event with appropriate signature
  - [ ] Handle different event types with proper parameters
- [ ] **Name Handling** (`cli/internal/typer/generator/kotlin/naming.go`):
  - [ ] Add `FormatMethodName()` - camelCase for methods
- [ ] **Templates**:
  - [ ] Create `rudderanalytics.tmpl` partial
  - [ ] Update `main.kt.tmpl` to include RudderAnalytics object

### Phase 4 Testing

- [ ] **Test** (`cli/internal/typer/generator/kotlin/generator_test.go`):
  - [ ] Update `TestGenerate()` to include RudderAnalytics methods
  - [ ] Build complete tracking plan with custom types and events
  - [ ] Validate RudderAnalytics object is generated
  - [ ] Verify methods exist for all events with correct signatures
  - [ ] Check proper type references in method parameters

**Deliverable**: Complete working generator with types and analytics methods

---

## Phase 5: Enums and Arrays

**Goal**: Handle enum properties and array types

### Enum and Array Support

- [ ] **Plan Models** (`cli/internal/typer/plan/`):
  - [ ] Add enum support to existing PropertyConfig
  - [ ] Add helper methods for enum detection
- [ ] **Context Types** (`cli/internal/typer/generator/kotlin/context.go`):
  - [ ] Define `KotlinEnum` and `KotlinEnumValue` structs
  - [ ] Add Enums slice to `KotlinContext`
- [ ] **Generator Logic** (`cli/internal/typer/generator/kotlin/generator.go`):
  - [ ] Generate KotlinEnum for properties with enum config
  - [ ] Handle array types with `List<T>` syntax
  - [ ] Support arrays of custom types
- [ ] **Name Handling** (`cli/internal/typer/generator/kotlin/naming.go`):
  - [ ] Add `FormatEnumValue()` - UPPER_SNAKE_CASE
- [ ] **Templates**:
  - [ ] Create `enum.tmpl` partial
  - [ ] Update `main.kt.tmpl` to include enums section

### Phase 5 Testing

- [ ] **Test** (`cli/internal/typer/generator/kotlin/generator_test.go`):
  - [ ] Update `TestGenerate()` to include enums and arrays
  - [ ] Build tracking plan with enum properties and array types
  - [ ] Validate enum classes are generated
  - [ ] Verify array types use proper List<T> syntax
  - [ ] Check enum values in properties reference generated enums

**Deliverable**: Full generator supporting enums and arrays

---

## Phase 6: Variants (Sealed Classes)

**Goal**: Support variants with sealed classes

### Variant Support

- [ ] **Plan Models** (`cli/internal/typer/plan/`):
  - [ ] Add `Variant` and `VariantCase` types
  - [ ] Update `CustomType` to support variants
- [ ] **Context Types** (`cli/internal/typer/generator/kotlin/context.go`):
  - [ ] Define `KotlinSealedClass` and `KotlinSealedClassCase` structs
  - [ ] Add SealedClasses slice to `KotlinContext`
- [ ] **Generator Logic** (`cli/internal/typer/generator/kotlin/generator.go`):
  - [ ] Detect custom types with variants
  - [ ] Generate sealed class with discriminator
  - [ ] Generate case classes for each variant
- [ ] **Templates**:
  - [ ] Create `sealedclass.tmpl` partial
  - [ ] Update `main.kt.tmpl` to include sealed classes

### Phase 6 Testing

- [ ] **Test** (`cli/internal/typer/generator/kotlin/generator_test.go`):
  - [ ] Update `TestGenerate()` to include variants
  - [ ] Build tracking plan with variant custom types
  - [ ] Validate sealed classes are generated
  - [ ] Verify case classes with proper discriminator values
  - [ ] Check serialization annotations

**Deliverable**: Complete generator with variant support

---

## Phase 7: Edge Cases and Polish

**Goal**: Handle reserved keywords, collisions, and special characters

### Edge Case Handling

- [ ] **Name Handling** (`cli/internal/typer/generator/kotlin/naming.go`):
  - [ ] Create comprehensive Kotlin reserved keywords list
  - [ ] Implement keyword escaping strategies
  - [ ] Enhance collision handler with better strategies
- [ ] **Generator Logic** (`cli/internal/typer/generator/kotlin/generator.go`):
  - [ ] Handle properties starting with numbers
  - [ ] Process special characters in names and comments
  - [ ] Add comprehensive error handling

## Implementation Notes

### Key Design Principles

1. **Separation of Concerns**: Clear boundaries between tracking plan model, template context, and generated code
2. **Extensibility**: Strategy pattern enables easy addition of new platforms
3. **Template-based**: Go templates provide flexibility and maintainability
4. **Name Safety**: Dedicated registry handles collisions and reserved words
5. **Comprehensive Testing**: Every major component must have thorough unit tests with good coverage
6. **Test-Driven Development**: Tests should be written alongside or before implementation

### Testing Requirements

**IMPORTANT**: Every major component introduced in this implementation must include comprehensive unit tests:

- **Core Infrastructure**: NameRegistry, GeneratorStrategy interface, and utility functions
- **Context Types**: All Kotlin context structs and their methods
- **Name Handling**: All naming functions and collision handlers
- **Generator Logic**: All processing functions and generation methods
- **Templates**: Template rendering and context integration

Tests should:

- Cover both success and error scenarios
- Test edge cases and boundary conditions
- Use meaningful test data that reflects real-world usage
- Be placed in `*_test` packages for proper encapsulation
- Follow Go testing conventions and use testify/assert for clarity

### Implementation Dependencies

- The NameRegistry must be implemented first as it's used by all generators
- Plan model enhancements should be completed before generator work
- Template utilities should be built alongside template development
- Testing should be developed incrementally with each feature

### Success Criteria

- [ ] Generate valid, compilable Kotlin code from any valid tracking plan
- [ ] Handle all edge cases (reserved words, collisions, special characters)
- [ ] Support all tracking plan features (custom types, variants, events)
- [ ] Demonstrate extensibility with multiple platform generators
- [ ] Comprehensive test coverage with single integration test
