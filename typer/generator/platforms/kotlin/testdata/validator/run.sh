#!/bin/bash

# Create source directories
mkdir -p /app/src/main/kotlin
mkdir -p /app/src/test/kotlin

# Copy main Kotlin files from external volume, preserving directory structure
if [[ -d /app/external/main/kotlin ]]; then
    cp -r /app/external/main/kotlin/* /app/src/main/kotlin/ 2>/dev/null || true
fi

# Copy test files from external volume, preserving directory structure
if [[ -d /app/external/test/kotlin ]]; then
    cp -r /app/external/test/kotlin/* /app/src/test/kotlin/ 2>/dev/null || true
fi

# Run tests
gradle test --no-daemon --warning-mode all