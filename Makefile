.PHONY: build test clean install lint fmt help

# Build variables
BINARY_NAME=regula
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Go commands
GO=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOINSTALL=$(GO) install
GOMOD=$(GO) mod
GOFMT=gofmt

## help: Print this help message
help:
	@echo "Regula - Automated Regulation Mapper"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'

## build: Build the regula binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/regula

## test: Run all tests
test:
	$(GOTEST) -v ./...

## test-cover: Run tests with coverage
test-cover:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## install: Install regula to $GOPATH/bin
install:
	$(GOINSTALL) $(LDFLAGS) ./cmd/regula

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## lint: Run linter
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GOFMT) -w -s .

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## deps: Download dependencies
deps:
	$(GOMOD) download

## run: Build and run with example command
run: build
	./$(BINARY_NAME) --help

## example-init: Run init example
example-init: build
	./$(BINARY_NAME) init gdpr-analysis

## example-query: Run query example
example-query: build
	./$(BINARY_NAME) query "SELECT ?p WHERE { ?p rdf:type reg:Provision }"

## example-impact: Run impact example
example-impact: build
	./$(BINARY_NAME) impact --provision "GDPR:Art17" --change amend

## example-simulate: Run simulate example
example-simulate: build
	./$(BINARY_NAME) simulate --scenario examples/gdpr/consent-withdrawal.yaml

# Default target
.DEFAULT_GOAL := help
