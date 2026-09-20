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
# The catalog documents experimental rETL kinds too, so their flags default on
# here, which is how a plain local run generates the same catalog CI does. CI
# sets them explicitly; an explicit environment value still wins.
gen-rule-docs: ## Generate the validation rule documentation artifact
	RUDDERSTACK_CLI_EXPERIMENTAL=$${RUDDERSTACK_CLI_EXPERIMENTAL:-true} \
	RUDDERSTACK_X_RETL_TABLE_SUPPORT=$${RUDDERSTACK_X_RETL_TABLE_SUPPORT:-true} \
	$(GO) run ./cli/cmd/gen-rule-docs --output-dir $(RULE_DOCS_OUTPUT_DIR)

.PHONY: test
test: ## Run all unit tests (excluding e2e)
	@go test --race --covermode=atomic --coverprofile=coverage-unit.out $(shell go list ./... | grep -v '/cli/tests$$')

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

##@ Demos

# Matches api/client/client.go's BASE_URL — the host the CLI silently falls
# back to (with whatever real token ~/.rudder/config.json holds) whenever
# RUDDERSTACK_API_URL is unset. The e2e suite's first act is a workspace-
# wiping `destroy --confirm=false`, so demo-record refuses rather than risk
# aiming that at production.
PROD_API_HOST := api.rudderstack.com

DEMO_TEST ?= TestAccountsApply
DEMO_PROFILE ?= mini
DEMO_OUT ?= demos
DEMO_JOURNAL ?= $(DEMO_OUT)/$(DEMO_TEST).jsonl
DEMO_EVENTS ?= $(DEMO_OUT)/$(DEMO_TEST).events.json

.PHONY: demo-record
demo-record: ## Run one e2e test with journaling on (TEST=..., PROFILE=mini|cloud)
	@test -f ./demos/profiles/$(DEMO_PROFILE).env || \
		{ echo "refusing: no such profile ./demos/profiles/$(DEMO_PROFILE).env"; exit 1; }
	@set -a; . ./demos/profiles/$(DEMO_PROFILE).env; set +a; \
	if [ -z "$$RUDDERSTACK_API_URL" ]; then \
		echo "refusing: RUDDERSTACK_API_URL is unset, so the CLI would target $(PROD_API_HOST)."; \
		echo "  The e2e suite runs 'destroy --confirm=false' first — it wipes the target workspace."; \
		echo "  Source a profile: . ./demos/profiles/mini.env"; \
		exit 1; \
	fi; \
	case "$$RUDDERSTACK_API_URL" in \
		*$(PROD_API_HOST)*) \
			echo "refusing: RUDDERSTACK_API_URL points at production, and this target wipes its workspace."; \
			exit 1 ;; \
	esac; \
	mkdir -p $(DEMO_OUT); \
	rm -f $(DEMO_JOURNAL); \
	RUDDER_DEMO_JOURNAL=$(abspath $(DEMO_JOURNAL)) \
	$(GO) test -json -count=1 -timeout 20m ./cli/tests -run '^$(DEMO_TEST)$$' -v > $(DEMO_EVENTS) || \
		{ echo "e2e run failed — see $(DEMO_EVENTS)"; exit 1; }
	@echo "journal: $(DEMO_JOURNAL)"
	@echo "events:  $(DEMO_EVENTS)"

.PHONY: demo-generate
demo-generate: ## Generate demos/<Test>/ from the last demo-record
	@set -a; . ./demos/profiles/$(DEMO_PROFILE).env; set +a; \
	bin="$$(grep -o '"argv":\["[^"]*"' $(DEMO_JOURNAL) | sed -E 's/.*\["//;s/"$$//' | grep -E '/rudder-cli(\.exe)?$$' | head -1)"; \
	if [ -z "$$bin" ]; then \
		echo "refusing: no rudder-cli invocation found in $(DEMO_JOURNAL) to discover the CLI binary from"; \
		exit 1; \
	fi; \
	$(GO) run ./cli/tests/demo/cmd/demogen \
		-journal $(DEMO_JOURNAL) \
		-events $(DEMO_EVENTS) \
		-out $(DEMO_OUT) \
		-repo-root . \
		-bin "$$bin" \
		-profile "$$RUDDER_DEMO_PROFILE" \
		-backend-kind "$$RUDDER_DEMO_BACKEND_KIND" \
		-api-url "$$RUDDERSTACK_API_URL"

.PHONY: demo
demo: demo-record demo-generate ## Record and generate one demo end to end

.PHONY: demo-check
demo-check: ## Regenerate demos/<TEST>/ from its committed journal and fail on drift (TEST=...)
	@DEMO_PROFILE=$(DEMO_PROFILE) ./scripts/demo-check.sh $(DEMO_TEST)

CASTS_OUT ?= casts

.PHONY: demo-cast
demo-cast: ## Record demos/<TEST>/demo.sh to casts/<TEST>.cast with provenance (TEST=...)
	@./scripts/demo-cast.sh $(DEMO_TEST) $(CASTS_OUT)

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
	go run ./cli/internal/typer/generator/platforms/typescript/testutils/identity_sections \
	  > cli/internal/typer/generator/platforms/typescript/testdata/IdentitySections.ts
	go run ./cli/internal/typer/generator/platforms/typescript/testutils/empty_identity \
	  > cli/internal/typer/generator/platforms/typescript/testdata/EmptyIdentity.ts

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
	cp cli/internal/typer/generator/platforms/typescript/testdata/IdentitySections.ts \
	   cli/internal/typer/generator/platforms/typescript/testdata/validator/src/RudderTyper/IdentitySections.ts
	cp cli/internal/typer/generator/platforms/typescript/testdata/EmptyIdentity.ts \
	   cli/internal/typer/generator/platforms/typescript/testdata/validator/src/RudderTyper/EmptyIdentity.ts
	cd cli/internal/typer/generator/platforms/typescript/testdata/validator && docker compose run --rm -T validator
