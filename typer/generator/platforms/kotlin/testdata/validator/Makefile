IMAGE_NAME = rudder-typer-kotlin-validator

.PHONY: build
build:
	docker build -t $(IMAGE_NAME) .

.PHONY: run
run: build
	docker run --rm -v "$(PWD)/src:/app/external" $(IMAGE_NAME)
