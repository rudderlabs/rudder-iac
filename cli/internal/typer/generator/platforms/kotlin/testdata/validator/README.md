# RudderTyper Kotlin Validator

A Docker-based Kotlin validation project that validates generated RudderTyper code using the actual RudderStack Kotlin SDK and JUnit tests.

## Overview

This project provides a containerized environment for validating that generated Kotlin code from RudderTyper compiles correctly and integrates properly with the RudderStack Kotlin SDK. It uses a custom SDK plugin to intercept events and validate their structure against expected values.

## Architecture

The validator uses a **plugin-based testing approach**:

1. **Real SDK Integration**: Uses the actual RudderStack Kotlin SDK (not mocks)
2. **Event Interception**: Custom `EventValidationPlugin` intercepts events sent through the SDK
3. **JUnit Tests**: Comprehensive test suite validates event structure and serialization
4. **Generated Code**: Loads externally generated RudderTyper code from test data

### Key Components

- **EventValidationPlugin**: SDK plugin that intercepts events and validates their structure
- **RudderTyperKotlinTests**: JUnit test suite exercising all generated RudderTyper methods
- **Generated Types**: RudderTyper-generated code loaded from external volume at runtime

## Features

- **Docker-based execution**: Consistent testing environment with no local JVM/Kotlin installation required
- **Real SDK validation**: Uses actual RudderStack Kotlin SDK to ensure generated code integrates correctly
- **Event interception**: Custom plugin captures and validates events without sending them to servers
- **Comprehensive tests**: Full coverage of all RudderStack event types (Track, Identify, Page, Screen, Group)
- **Type safety validation**: Ensures generated types compile and work with the SDK
- **Gradle build system**: Modern Kotlin project setup with JUnit 5

## Project Structure

```
├── src/
│   ├── main/kotlin/
│   │   └── com/rudderstack/ruddertyper/
│   │       └── Main.kt                    # Generated RudderTyper code (loaded from external volume)
│   └── test/kotlin/
│       ├── EventValidationPlugin.kt       # SDK plugin for event interception and validation
│       └── RudderTyperKotlinTests.kt      # JUnit test suite
├── build.gradle.kts                       # Gradle build configuration with SDK and test dependencies
├── settings.gradle.kts                    # Gradle settings
├── gradle.properties                      # Gradle properties
├── Dockerfile                             # Docker image with JDK 21 and Gradle
├── run.sh                                 # Container runtime script (copies external files and runs tests)
├── Makefile                               # Build and run commands
└── README.md                              # This file
```

## Requirements

- Docker

## Usage

### Build the Docker image

```bash
make build
```

### Run the validator

```bash
make run
```

This will:

1. Mount the current directory as `/app/external` in the container
2. Copy any external Kotlin files to the appropriate package directory
3. Execute the pre-compiled application

### Manual Docker commands

If you prefer not to use the Makefile:

```bash
# Build
docker build -t rudder-typer-kotlin-validator .

# Run
docker run --rm -v "$(pwd):/app/external" rudder-typer-kotlin-validator
```

## How It Works

### Event Validation Flow

1. **Setup**: Each test initializes the RudderStack SDK with the `EventValidationPlugin`
2. **Execution**: Test calls RudderTyper methods (e.g., `typer.trackUserSignedUp(...)`)
3. **Interception**: `EventValidationPlugin` intercepts the event before it's sent
4. **Validation**: Plugin validates event type, properties, and serialization against expected values
5. **Assertion**: JUnit assertions ensure the event matches expectations

### EventValidationPlugin

The custom SDK plugin implements the `Plugin` interface and intercepts events during the `OnProcess` phase:

```kotlin
override suspend fun intercept(event: Event): Event? {
    received.add(event)
    return null  // Block event from being sent (validation only)
}
```

**Validation Methods**:
- `validateCount(expected)`: Ensures the expected number of events were received (with timeout)
- `validateNext(expectedValidation)`: Validates the next event against expected values

### External Generated Code

The validator expects generated RudderTyper code from an external Docker volume mounted at `/app/external`. Running `make typer-kotlin-validate` copies the test data from [cli/internal/typer/generator/platforms/kotlin/testdata/Main.kt](../Main.kt) - the same code used in Go unit tests - ensuring consistency between static code generation tests and runtime validation.

## Dependencies

- **Kotlin**: 1.9.22
- **JDK**: 17
- **RudderStack Kotlin SDK**: 1.1.0
- **Gradle**: 8.5

## Writing Tests

Tests follow a standard pattern:

```kotlin
@Test
fun testTrackEvent() {
    // 1. Call RudderTyper method
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            email = "user@example.com"
        )
    )

    // 2. Flush events to trigger plugin interception
    analytics.flush()

    // 3. Validate event count
    validations.validateCount(1)

    // 4. Validate event structure
    validations.validateNext(
        EventValidation.TrackValidation(
            name = "User Signed Up",
            properties = buildJsonObject {
                put("active", true)
                put("email", "user@example.com")
            }
        )
    )
}
```

**Important**: Expected properties must match the actual serialized JSON structure, including:
- Proper property names as they appear in the tracking plan
- Nested objects for custom types
- Correct JSON types (strings, numbers, booleans, objects, arrays)

## Docker Build Process

The Dockerfile creates a containerized testing environment:

1. **Base image**: Uses `eclipse-temurin:21-jdk` for JDK 21 support
2. **Gradle installation**: Installs Gradle 8.4 for building and testing
3. **Dependency pre-fetching**: Downloads SDK and test dependencies during image build
4. **Runtime**: `run.sh` copies generated code from external volume and executes tests
