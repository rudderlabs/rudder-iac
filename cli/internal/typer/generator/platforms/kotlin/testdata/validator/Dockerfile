# Use a lightweight base image with JDK
FROM eclipse-temurin:21-jdk@sha256:ec2005c536f3661c6ef1253292c9c623e582186749a3ef2ed90903d1aaf74640

# Install Gradle
ENV GRADLE_VERSION=8.4
RUN apt-get update && \
    apt-get --no-install-recommends install -y curl unzip && \
    curl --proto "=https" --tlsv1.2 -sSf -L https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip -o gradle.zip && \
    unzip gradle.zip && \
    mv gradle-${GRADLE_VERSION} /opt/gradle && \
    ln -s /opt/gradle/bin/gradle /usr/local/bin/gradle && \
    rm gradle.zip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create a non-root user for running the application
RUN groupadd -r gradle && useradd -r -g gradle -m gradle

# Set work directory
WORKDIR /app

# Copy build files
COPY build.gradle.kts .
COPY settings.gradle.kts .
COPY gradle.properties .

# Copy run script
COPY run.sh .
RUN chmod +x run.sh

# Change ownership of app directory and gradle home to non-root user after building
RUN chown -R gradle:gradle /app && \
    chown -R gradle:gradle /home/gradle

# Switch to non-root user for running the application
USER gradle

RUN gradle dependencies --no-daemon

# Run the application
CMD ["./run.sh"]
