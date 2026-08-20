# Build stage
FROM golang:1.25.11-alpine@sha256:523c3effe300580ed375e43f43b1c9b091b68e935a7c3a92bfcc4e7ed55b18c2 AS builder
ARG VERSION=0.0.0
ARG TELEMETRY_WRITE_KEY=""
ARG TELEMETRY_DATAPLANE_URL=""
RUN apk add --no-cache make
WORKDIR /build
COPY go.mod go.sum ./
RUN set -e; \
    for attempt in 1 2 3; do \
        go mod download && exit 0; \
        if [ "$attempt" -eq 3 ]; then exit 1; fi; \
        sleep $((attempt * 5)); \
    done
COPY . .
RUN CGO_ENABLED=0 make build VERSION=$VERSION

# Final stage - using alpine for sh support
# sh is required to be present in the image because the image
# should be usable as part of a GitLab CI/CD pipeline.
FROM alpine:latest@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412
COPY --from=builder /build/bin/rudder-cli /usr/local/bin/rudder-cli
# Create config directory
VOLUME /root/.rudder
ENTRYPOINT ["/usr/local/bin/rudder-cli"]
