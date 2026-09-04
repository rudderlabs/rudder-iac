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
gen-rule-docs: ## Generate the validation rule documentation artifact
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
	cd cli/internal/typer/generator/platforms/kotlin/testdata/validator && make run

.PHONY: typer-kotlin-update-testdata
typer-kotlin-update-testdata: ## Update test data for Kotlin code generation
	go run cli/internal/typer/generator/platforms/kotlin/testutils/generate_reference_plan.go

.PHONY: typer-swift-update-testdata
typer-swift-update-testdata: ## Update test data for Swift code generation
	go run cli/internal/typer/generator/platforms/swift/testutils/generate_reference_plan.go \
	  > cli/internal/typer/generator/platforms/swift/testdata/RudderTyper.swift

.PHONY: typer-typescript-update-testdata
typer-typescript-update-testdata: ## Update test data for TypeScript code generation
	go run cli/internal/typer/generator/platforms/typescript/testutils/generate_reference_plan.go \
	  > cli/internal/typer/generator/platforms/typescript/testdata/RudderTyper.ts

.PHONY: typer-swift-validate
typer-swift-validate: ## Validate generated Swift code against the RudderStack Swift SDK
	mkdir -p cli/internal/typer/generator/platforms/swift/testdata/validator/Sources/RudderTyper
	cp cli/internal/typer/generator/platforms/swift/testdata/RudderTyper.swift \
	   cli/internal/typer/generator/platforms/swift/testdata/validator/Sources/RudderTyper/RudderTyper.swift
	cd cli/internal/typer/generator/platforms/swift/testdata/validator && swift test --disable-swift-testing

.PHONY: typer-typescript-validate
typer-typescript-validate: ## Validate generated TypeScript code against the RudderStack JS SDK
	mkdir -p cli/internal/typer/generator/platforms/typescript/testdata/validator/src/RudderTyper
	cp cli/internal/typer/generator/platforms/typescript/testdata/RudderTyper.ts \
	   cli/internal/typer/generator/platforms/typescript/testdata/validator/src/RudderTyper/RudderTyper.ts
	cd cli/internal/typer/generator/platforms/typescript/testdata/validator && docker compose run --rm -T validator

# Directory of a rudder-data-gov checkout to verify `typer generate --local` against.
# It intentionally holds data-graphs/ and transformations/ next to its catalog —
# kinds the local typer does not register — so this exercises the skip-unknown-kinds path.
DATAGOV_DIR ?=
DATAGOV_TRACKING_PLAN ?= webapp
TS_VALIDATOR_DIR = cli/internal/typer/generator/platforms/typescript/testdata/validator

.PHONY: typer-verify-datagov
typer-verify-datagov: build ## Generate a typed TS client from a rudder-data-gov checkout via --local and typecheck it against the JS SDK (set DATAGOV_DIR)
	@test -n "$(DATAGOV_DIR)" || { echo "DATAGOV_DIR is required, e.g. make typer-verify-datagov DATAGOV_DIR=../rudder-data-gov"; exit 1; }
	mkdir -p $(TS_VALIDATOR_DIR)/src/RudderTyper
	RUDDERSTACK_CLI_EXPERIMENTAL=true RUDDERSTACK_X_LOCAL_TYPER=true \
	  bin/rudder-cli typer generate --local \
	    --location $(DATAGOV_DIR) \
	    --tracking-plan-id $(DATAGOV_TRACKING_PLAN) \
	    --platform typescript \
	    -o $(TS_VALIDATOR_DIR)/src/RudderTyper
	# typecheck the generated file only (tsconfig.gen-only) — the reference-plan
	# vitest suite is written against a different plan and would not compile here.
	cd $(TS_VALIDATOR_DIR) && docker compose run --rm -T validator \
	  sh -c "npm ci && npx tsc -p tsconfig.gen-only.json"
