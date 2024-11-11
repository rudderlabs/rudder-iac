.PHONY: all clean build

VERSION ?= 0.1

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/rudder-cli ./cli

all: build

clean:
	rm -rf bin
