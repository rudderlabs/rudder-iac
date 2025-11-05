#!/bin/bash

# Copy Types.kt from external volume if it exists
cp /app/external/* /app/src/main/kotlin/ 2>/dev/null || true

gradle build --no-daemon

# Run the fat jar
java -jar build/libs/*.jar