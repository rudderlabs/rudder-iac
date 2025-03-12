# Build stage
FROM golang:1.23-alpine AS builder
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
FROM alpine:latest
COPY --from=builder /build/bin/rudder-cli /usr/local/bin/rudder-cli
# Create config directory
VOLUME /root/.rudder
ENTRYPOINT ["/usr/local/bin/rudder-cli"]
