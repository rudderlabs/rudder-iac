VERSION ?= 0.1
REGISTRY ?= rudderlabs
IMAGE_NAME ?= rudder-cli
TELEMETRY_WRITE_KEY ?= ""
TELEMETRY_DATAPLANE_URL ?= ""
GO=go
GOLANGCI=github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0

.PHONY: all
all: build

.PHONY: help
help: ## Show the available commands
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' ./Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: lint
lint: ## Run linters on all go files
	$(GO) run $(GOLANGCI) run -v

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

RULE_DOCS_OUTPUT_DIR ?= docs/generated

.PHONY: gen-rule-docs
# The catalog documents the complete rule set, so generation turns on the
# experimental features whose rules ship with authored fragments; without them
# those fragments read as stale. That is also how a plain local run generates
# the same catalog CI does. Defaulted rather than pinned so CI, which sets them
# explicitly, still wins.
gen-rule-docs: ## Generate the validation rule documentation artifact
	RUDDERSTACK_CLI_EXPERIMENTAL=$${RUDDERSTACK_CLI_EXPERIMENTAL:-true} \
	RUDDERSTACK_X_IMPORT_MERGE=$${RUDDERSTACK_X_IMPORT_MERGE:-true} \
	RUDDERSTACK_X_RETL_TABLE_SUPPORT=$${RUDDERSTACK_X_RETL_TABLE_SUPPORT:-true} \
	RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=$${RUDDERSTACK_X_RETL_CONNECTION_SUPPORT:-true} \
	$(GO) run ./cli/cmd/gen-rule-docs --output-dir $(RULE_DOCS_OUTPUT_DIR)

.PHONY: test
test: ## Run all unit tests (excluding e2e)
	@go test --race --covermode=atomic --coverprofile=coverage-unit.out $(shell go list ./... | grep -v /cli/tests)

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@go test --race --covermode=atomic --coverprofile=coverage-e2e.out -timeout 20m ./cli/tests/...  -v

.PHONY: test-it
test-it: ## Run all test, including integration tests
	go test -tags integrationtest ./...

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg TELEMETRY_WRITE_KEY=$(TELEMETRY_WRITE_KEY) \
		--build-arg TELEMETRY_DATAPLANE_URL=$(TELEMETRY_DATAPLANE_URL) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		-f cli/Dockerfile .

.PHONY: test-all
test-all: test test-e2e ## Run all unit and end-to-end tests

.PHONY: typer-kotlin-validate
typer-kotlin-validate: ## Validate generated Kotlin code inside a Kotlin project
	cd typer/generator/platforms/kotlin/testdata/validator && make run

.PHONY: typer-kotlin-update-testdata
typer-kotlin-update-testdata: ## Update test data for Kotlin code generation
	go run typer/generator/platforms/kotlin/testutils/generate_reference_plan.go

.PHONY: typer-swift-update-testdata
typer-swift-update-testdata: ## Update test data for Swift code generation
	go run typer/generator/platforms/swift/testutils/generate_reference_plan.go \
	  > typer/generator/platforms/swift/testdata/RudderTyper.swift

.PHONY: typer-typescript-update-testdata
typer-typescript-update-testdata: ## Update test data for TypeScript code generation
	go run typer/generator/platforms/typescript/testutils/generate_reference_plan.go \
	  > typer/generator/platforms/typescript/testdata/RudderTyper.ts
	go run ./typer/generator/platforms/typescript/testutils/identity_sections \
	  > typer/generator/platforms/typescript/testdata/IdentitySections.ts
	go run ./typer/generator/platforms/typescript/testutils/empty_identity \
	  > typer/generator/platforms/typescript/testdata/EmptyIdentity.ts

.PHONY: typer-swift-validate
typer-swift-validate: ## Validate generated Swift code against the RudderStack Swift SDK
	mkdir -p typer/generator/platforms/swift/testdata/validator/Sources/RudderTyper
	cp typer/generator/platforms/swift/testdata/RudderTyper.swift \
	   typer/generator/platforms/swift/testdata/validator/Sources/RudderTyper/RudderTyper.swift
	cd typer/generator/platforms/swift/testdata/validator && swift test --disable-swift-testing

.PHONY: typer-typescript-validate
typer-typescript-validate: ## Validate generated TypeScript code against the RudderStack JS SDK
	mkdir -p typer/generator/platforms/typescript/testdata/validator/src/RudderTyper
	cp typer/generator/platforms/typescript/testdata/RudderTyper.ts \
	   typer/generator/platforms/typescript/testdata/validator/src/RudderTyper/RudderTyper.ts
	cp typer/generator/platforms/typescript/testdata/IdentitySections.ts \
	   typer/generator/platforms/typescript/testdata/validator/src/RudderTyper/IdentitySections.ts
	cp typer/generator/platforms/typescript/testdata/EmptyIdentity.ts \
	   typer/generator/platforms/typescript/testdata/validator/src/RudderTyper/EmptyIdentity.ts
	cd typer/generator/platforms/typescript/testdata/validator && docker compose run --rm -T validator
