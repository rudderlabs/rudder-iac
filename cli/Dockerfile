# Build stage
FROM golang:1.24.9-alpine@sha256:8f8959f38530d159bf71d0b3eb0c547dc61e7959d8225d1599cf762477384923 AS builder
ARG VERSION=0.0.0
ARG TELEMETRY_WRITE_KEY=""
ARG TELEMETRY_DATAPLANE_URL=""
RUN apk add --no-cache make
WORKDIR /build
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
