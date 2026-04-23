.PHONY: build install test cover vet fmt lint clean snapshot release-dry tidy help

BINARY := bmlt
PKG    := github.com/bmlt-enabled/bmlt-cli

# Local dev metadata so `make build` produces a labelled binary.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
  -X main.version=$(VERSION) \
  -X main.commit=$(COMMIT) \
  -X main.date=$(DATE)

help: ## Show this help
	@awk 'BEGIN{FS=":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Compile bmlt for the host platform
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: ## Install bmlt into $GOBIN
	go install -trimpath -ldflags "$(LDFLAGS)" .

test: ## Run unit tests with race detector
	go test -race ./...

cover: ## Run tests with coverage and open HTML report
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser."

vet: ## go vet
	go vet ./...

fmt: ## gofmt -w
	gofmt -w .

lint: vet ## vet + gofmt -l (CI-style)
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt issues:"; echo "$$out"; exit 1; fi

tidy: ## go mod tidy
	go mod tidy

snapshot: ## Build all release artifacts locally without publishing
	goreleaser release --snapshot --clean

release-dry: ## Validate goreleaser config + dry run
	goreleaser check
	goreleaser release --snapshot --clean --skip=publish

clean: ## Remove build artifacts
	rm -rf $(BINARY) dist/ coverage.out coverage.html
