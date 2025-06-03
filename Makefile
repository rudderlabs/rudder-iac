VERSION ?= 0.1
REGISTRY ?= rudderlabs
IMAGE_NAME ?= rudder-cli
TELEMETRY_WRITE_KEY ?= ""
TELEMETRY_DATAPLANE_URL ?= ""

.PHONY: all
all: build

.PHONY: help
help: ## Show the available commands
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' ./Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build:
	go build \
		-ldflags "\
			-s -w \
			-X 'main.version=$(VERSION)' \
			-X 'github.com/rudderlabs/rudder-iac/cli/internal/config.TelemetryWriteKey=$(TELEMETRY_WRITE_KEY)' \
			-X 'github.com/rudderlabs/rudder-iac/cli/internal/config.TelemetryDataplaneURL=$(TELEMETRY_DATAPLANE_URL)' \
		" \
		-o bin/rudder-cli \
		./cli/cmd/rudder-cli

.PHONY: clean
clean:
	rm -rf bin

.PHONY: test
test: ## Run all unit tests
	go test --race --covermode=atomic --coverprofile=coverage.out ./...

.PHONY: test-it
test-it: ## Run all test, including integration tests
	go test -tags integrationtest ./...

# Run all E2E tests in the tests directory with verbose output
.PHONY: test-e2e
test-e2e:
	go test -v ./tests/...

# (You can add filtering by test name or file in the future if needed)

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg TELEMETRY_WRITE_KEY=$(TELEMETRY_WRITE_KEY) \
		--build-arg TELEMETRY_DATAPLANE_URL=$(TELEMETRY_DATAPLANE_URL) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		-f cli/Dockerfile .
