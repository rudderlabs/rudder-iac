VERSION ?= 0.1
REGISTRY ?= rudderlabs
IMAGE_NAME ?= rudder-cli

.PHONY: all
all: build

.PHONY: help
help: ## Show the available commands
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' ./Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/rudder-cli ./cli/cmd/rudder-cli

.PHONY: clean
clean:
	rm -rf bin

.PHONY: test
test: ## Run all unit tests
	go test --race --covermode=atomic --coverprofile=coverage.out ./...

.PHONY: test-it
test-it: ## Run all test, including integration tests
	go test -tags integrationtest ./...

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		-t $(REGISTRY)/$(IMAGE_NAME):latest \
		-f cli/Dockerfile .

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION)
	docker push $(REGISTRY)/$(IMAGE_NAME):latest
