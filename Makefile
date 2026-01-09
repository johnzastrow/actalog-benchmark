.PHONY: build clean test run install

BINARY_NAME=actalog-bench
VERSION ?= 0.7.0

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/actalog-bench

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/actalog-bench

clean:
	rm -rf bin/

test:
	go test -v ./...

run: build
	./bin/$(BINARY_NAME)

lint:
	golangci-lint run

fmt:
	go fmt ./...

deps:
	go mod download
	go mod tidy
