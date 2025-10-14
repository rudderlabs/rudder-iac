# Use a lightweight base image with JDK
FROM openjdk:17-slim

# Install Gradle
ENV GRADLE_VERSION=8.4
RUN apt-get update && \
    apt-get install -y curl unzip && \
    curl -L https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip -o gradle.zip && \
    unzip gradle.zip && \
    mv gradle-${GRADLE_VERSION} /opt/gradle && \
    ln -s /opt/gradle/bin/gradle /usr/local/bin/gradle && \
    rm gradle.zip

# Set work directory
WORKDIR /app

# Copy build files
COPY build.gradle.kts .
COPY settings.gradle.kts .
COPY gradle.properties .

# Copy source code
RUN mkdir -p ./src/main/kotlin

# Copy run script
COPY run.sh .
RUN chmod +x run.sh

# Build the application
RUN gradle dependencies --no-daemon

# Run the application
CMD ["./run.sh"]
