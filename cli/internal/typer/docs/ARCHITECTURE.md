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
- **Kotlin Generator**:
  ```go
  func Generate(plan *plan.TrackingPlan) ([]*core.File, error)
  ```

### Template System

- **Purpose**: Handles code generation using Go templates
- **Current Implementation**: Embedded templates in platform packages
- **Features**:
  - Go template engine with embed directives
  - Sub-template composition
  - Context-driven rendering
- **Example**: `typealias.tmpl` for Kotlin type aliases

### Components Not Yet Implemented

- **PlanProvider**: Abstraction for tracking plan retrieval (planned)
- **FileManager**: File system operations handler (planned)
- **RudderTyper Orchestrator**: Main coordination component (planned)
- **TyperCommand**: CLI integration (planned)

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

**Template Design**:

- Specific contexts for different constructs (KotlinDataClass, KotlinTypeAlias)
- Pre-processed values (no escaping or formatting in templates)
- Minimal conditional logic

**Current Implementation**: Located in `cli/internal/typer/generator/platforms/kotlin/context.go`

## Design Principles

### Separation of Concerns

- TrackingPlan: Pure domain representation
- TemplateContext: Pure code construct representation
- Generators: Business logic transformation layer
- Templates: Minimal formatting logic

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

## Implementation Guidelines

### Adding New Platforms

1. Create a new package under `cli/internal/typer/generator/platforms/{platform}/`
2. Define platform-specific context types (like `KotlinContext`)
3. Implement a `Generate` function following the current pattern
4. Create embedded Go templates using `//go:embed`
5. Implement platform-specific collision handlers and naming functions
6. Add template processing functions

### Extending TrackingPlan Model

- Add new fields only if they represent domain concepts
- Maintain backwards compatibility
- Avoid derived or computed fields
- Document business justification

### Template Development

- Keep logic minimal
- Pre-process all values in generators
- Use specific context types for different constructs
- Avoid raw tracking plan access
