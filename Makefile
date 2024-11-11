VERSION ?= 0.1

.PHONY: all
all: build

.PHONY: help
help: ## Show the available commands
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' ./Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/rudder-cli ./cli

.PHONY: clean
clean:
	rm -rf bin

.PHONY: test
test: ## Run all unit tests
	go test ./...

.PHONY: test-it
test-it: ## Run all test, including integration tests
	go test -tags integrationtest ./...
