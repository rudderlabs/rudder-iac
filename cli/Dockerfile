# syntax=docker/dockerfile:1
# golang:1.24.9-alpine3.21
FROM golang:1.24.9-alpine3.21@sha256:4770ea1a103b3dc1a6e0c7c8d7a7a91d08ab0d4302f32256bc7b42d06f892103 AS builder
ARG VERSION=0.0.0
ARG TELEMETRY_WRITE_KEY=""
ARG TELEMETRY_DATAPLANE_URL=""
RUN apk add --no-cache --update make tzdata ca-certificates
ENV CGO_ENABLED=0
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build VERSION=$VERSION

# Final stage - using alpine for sh support
# sh is required to be present in the image because the image
# should be usable as part of a GitLab CI/CD pipeline.
# alpine:3.21
FROM alpine:3.21@sha256:c3f8e73fdb79deaebaa2037150150191b9dcbfba68b4a46d70103204c53f4709
RUN apk --no-cache upgrade && apk --no-cache add ca-certificates tzdata
RUN addgroup -g 10001 -S appgroup && adduser -u 10001 -S appuser -G appgroup -h /home/appuser
COPY --link --from=builder /build/bin/rudder-cli /usr/local/bin/rudder-cli
VOLUME /home/appuser/.rudder
USER 10001
ENTRYPOINT ["/usr/local/bin/rudder-cli"]
