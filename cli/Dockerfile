# Build stage
FROM golang:1.23-alpine AS builder
ARG VERSION=0.0.0
ARG TELEMETRY_WRITE_KEY=""
ARG TELEMETRY_DATAPLANE_URL=""
RUN apk add --no-cache make
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 make build VERSION=$VERSION

# Final stage - using scratch (empty) base image
FROM scratch
COPY --from=builder /build/bin/rudder-cli /rudder-cli
# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Create config directory
VOLUME /root/.rudder
ENTRYPOINT ["/rudder-cli"]
