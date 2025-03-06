# Build stage
FROM golang:1.23-alpine AS builder
ARG VERSION=0.0.0
RUN apk add --no-cache make
WORKDIR /build
COPY . .
RUN make build VERSION=$VERSION

# Final stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /build/bin/rudder-cli /usr/local/bin/
VOLUME /root/.rudder
ENTRYPOINT ["rudder-cli"]
