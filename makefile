GO ?= go
TOOLS = $(CURDIR)/.tools

run: check

$(TOOLS):
	@mkdir -p $@
$(TOOLS)/%: | $(TOOLS)
	@GOBIN=$(TOOLS) go install $(PACKAGE)

GOLANGCI_LINT = $(TOOLS)/golangci-lint
$(GOLANGCI_LINT): PACKAGE=github.com/golangci/golangci-lint/cmd/golangci-lint@latest

MOCKGEN = $(TOOLS)/mockgen
$(MOCKGEN): PACKAGE=github.com/golang/mock/mockgen@latest

GOVULNCHECK = $(TOOLS)/govulncheck
$(GOVULNCHECK): PACKAGE=golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: govulncheck
govulncheck: | $(GOVULNCHECK)
	@echo "Running: govulncheck"
	@$(TOOLS)/govulncheck ./...

.PHONY: tools
tools: $(GOLANGCI_LINT) $(JUNIT) $(MOCKGEN) $(GOVULNCHECK) $(SWAG)

.PHONY: tidy go-mod-tidy
tidy: go-mod-tidy
go-mod-tidy:
	@echo "Running: go mod tidy"
	@$(GO) mod tidy

.PHONY: lint golangci-lint golangci-lint-fix
lint: golangci-lint
golangci-lint-fix: ARGS=--fix
golangci-lint-fix: golangci-lint
golangci-lint: ARGS=--timeout=5m
golangci-lint: | $(GOLANGCI_LINT)
	@echo "Running: golangci-lint $(ARGS)"
	@$(TOOLS)/golangci-lint run $(ARGS)

.PHONY: check
check: go-mod-tidy golangci-lint test-race

.PHONY: generate
generate: | $(MOCKGEN)
	@PATH="$(PATH):$(TOOLS)" $(GO) generate ./...

.PHONY: check
check: go-mod-tidy golangci-lint test-race

VERSION := $(shell git rev-parse --short HEAD)
WEB_DIR = ./web
.PHONY: web-build
web-build:
	cd $(WEB_DIR) && npm i && npm run build
	
BUILD_OUPUT = ./bin/
.PHONY: build
build: web-build
build:
	@echo "Running: go build version=$(VERSION)"
	@mkdir -p bin/ \
	&&  $(GO) build -ldflags="-X main.Version=$(VERSION)" ${BUILD_TAGS} -o $(BUILD_OUPUT) ./...
	
.PHONY: test test-race
test-race: ARGS=-race
test-race: test
test:
	@echo "Running: go test $(ARGS)"
	@$(GO) test $(ARGS) ./...

.PHONY: bench
bench:
	@echo "Running: go test -bench ."
	@$(GO) test -run=nonthingplease -bench . ./...