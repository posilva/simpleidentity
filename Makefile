# SimpleIdentity Makefile
.PHONY: fmt build run test clean lint dev help setup  cover local-cover check health pprof install docker-run env-example start check

# Build variables
BINARY_NAME=simpleidentity
VERSION?=dev
GIT_COMMIT?=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/posilva/simpleidentity/cmd.Version=$(VERSION) -X github.com/posilva/simpleidentity/cmd.GitCommit=$(GIT_COMMIT) -X github.com/posilva/simpleidentity/cmd.BuildTime=$(BUILD_TIME)"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test -cover -timeout 50000ms -covermode=atomic
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Default target
help: ## Show this help message
	@echo "SimpleIdentity - Enterprise-grade identity management service"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/simpleidentity

build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/simpleidentity

build-docker: ## Build using Docker (for consistent builds)
	docker build -t simpleidentity:$(VERSION) .

run: ## Run the application in development mode
	$(GORUN) ./cmd/simpleidentity server --log-pretty --log-level debug

run-server: ## Run the server with production settings
	$(GORUN) ./cmd/simpleidentity server

test-unit: ## Run tests
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	$(GOTEST) -v -race -tags=integration ./test/integration/...

lint: ## Run linting
	golangci-lint run

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

dev: ## Start development server with auto-reload (requires air)
	air

check-health: ## Check health endpoints (server must be running)
	@echo "Checking health endpoints..."
	@curl -s http://localhost:8080/health | jq . || echo "Health endpoint failed"
	@curl -s http://localhost:8080/health/live | jq . || echo "Liveness endpoint failed"
	@curl -s http://localhost:8080/health/ready | jq . || echo "Readiness endpoint failed"

pprof: ## Open pprof in browser (server must be running)
	@echo "Opening pprof in browser..."
	@open http://localhost:6060/debug/pprof/ || xdg-open http://localhost:6060/debug/pprof/

install: build ## Install the binary to GOPATH/bin
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/

docker-run: ## Run in Docker container
	docker run -p 8080:8080 -p 6060:6060 -p 8090:8090 -p 9090:9090 simpleidentity:$(VERSION)

# Environment variable examples
env-example: ## Show environment variable examples
	@echo "Environment variable examples:"
	@echo "export SMPIDT_LOG_LEVEL=debug"
	@echo "export SMPIDT_LOG_PRETTY=true"
	@echo "export SMPIDT_HEALTH_ADDR=:8080"
	@echo "export SMPIDT_PPROF_ADDR=:6060"
	@echo "export SMPIDT_GRPC_ADDR=:9090"
	@echo "export SMPIDT_HTTP_ADDR=:8090"
	@echo "export SMPIDT_SHUTDOWN_TIMEOUT=30s"

# Quick start
start: deps build run ## Quick start: install deps, build, and run

setup:
	go install github.com/orlangure/gocovsh@latest

fmt: 
	go fmt ./...


cover:
	go tool cover -func=cover.out ./internal/...

local-cover:
	gocovsh coverage.out

check: lint test

test: test-unit test-integration
.DEFAULT_GOAL := help
