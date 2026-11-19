.PHONY: build run test clean install lint release

BINARY_NAME := cloudres
BUILD_DIR := build
GO := go
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

build:
	@echo "  • Building $(BINARY_NAME) ($(VERSION))..."
	@$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cloudres
	@echo "  • Built: $(BUILD_DIR)/$(BINARY_NAME)"

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	$(GO) test -v -race ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	@set -e; \
	dir="$(BINDIR)"; \
	if [ "$(origin PREFIX)/$(origin BINDIR)" = "file/file" ] && [ ! -w "$$dir" ]; then \
		dir="$(HOME)/.local/bin"; \
	fi; \
	mkdir -p "$$dir"; \
	install -m 0755 $(BUILD_DIR)/$(BINARY_NAME) "$$dir/$(BINARY_NAME)"; \
	echo "  • Installed: $$dir/$(BINARY_NAME)"

lint:
	golangci-lint run ./...

release:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64     ./cmd/cloudres
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64     ./cmd/cloudres
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64    ./cmd/cloudres
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64    ./cmd/cloudres
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/cloudres
