# RudderTyper Kotlin Validator

A Docker-based Kotlin validation project that compiles and executes Kotlin code with external type definitions from the `com.rudderstack.ruddertyper` package.

## Overview

This project provides a containerized environment for running Kotlin code with external type definitions that can be provided at runtime through Docker volumes. It includes the RudderStack Kotlin SDK as a dependency and supports dynamic loading of external Kotlin files.

## Features

- **Docker-based execution**: No need to install Kotlin/JVM locally
- **External type support**: Loads Types.kt from external volume at runtime
- **RudderStack integration**: Includes RudderStack Kotlin SDK dependency
- **Package structure**: Types use `com.rudderstack.ruddertyper` package
- **Gradle build system**: Modern Kotlin project setup with Gradle

## Project Structure

```
├── src/main/kotlin/
│   ├── Analytics.kt                   # Analytics mock file
│   └── RudderTyperKotlinValidator.kt  # Main application file
├── build.gradle.kts                   # Gradle build configuration
├── settings.gradle.kts                # Gradle settings
├── gradle.properties                  # Gradle properties
├── Dockerfile                         # Multi-stage Docker build
├── run.sh                            # Container runtime script
├── Makefile                          # Build and run commands
└── README.md                         # This file
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

## External Types

The project expects external type definitions to be added from an external Docker volume mounted at `/app/external`, with a package declaration of `com.rudderstack.ruddertyper`.

Running this using `make run` will copy the same test data from `cli/internal/typer/generator/platforms/kotlin/testdata/Main.kt` used in the Go tests, to ensure that generated Kotlin code compiles correctly.

## Dependencies

- **Kotlin**: 1.9.22
- **JDK**: 17
- **RudderStack Kotlin SDK**: 1.1.0
- **Gradle**: 8.5

## Development

The main application logic is in `src/main/kotlin/RudderTyperKotlinValidator.kt`. Modify this file to implement your validation logic using the external types.

## Docker Build Process

The project uses a multi-stage Docker build:

1. **Build stage**: Uses `gradle:8.5-jdk17` to compile the Kotlin code and resolve dependencies
2. **Runtime stage**: Uses `openjdk:17-jdk-slim` for a minimal runtime environment
3. **Volume mounting**: External files are copied at runtime via the `run.sh` script
