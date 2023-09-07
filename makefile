GO ?= go

VERSION := $(shell git rev-parse --short HEAD)
WEB_DIR = ./web
.PHONY: web-build
web-build:
	cd $(WEB_DIR) && npm run build
	
BUILD_OUPUT = ./bin/
.PHONY: build
build: web-build
build:
	@echo "Running: go build version=$(VERSION)"
	@mkdir -p bin/ \
	&&  $(GO) build -ldflags="-X main.Version=$(VERSION)" ${BUILD_TAGS} -o $(BUILD_OUPUT) ./...
	

