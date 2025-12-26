.PHONY: build test clean fmt vet check dev install uninstall network-clean help

BINARY_NAME := minicontainer
VERSION := 0.1.0
BUILD_DIR := .
GO_FILES := $(shell find . -name '*.go' -type f)

# Default target
all: build

# Build the binary
build:
	go build -o $(BINARY_NAME) .

# Build with version info
build-release:
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY_NAME) .

# Build with race detector (for development/debugging)
dev:
	go build -race -o $(BINARY_NAME) .

# Run tests (requires root)
test:
	sudo go test ./... -v

# Run tests with coverage
test-coverage:
	sudo go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	go clean

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run all checks
check: fmt vet build clean
	@echo "All checks passed!"

# Install to /usr/local/bin
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

# Uninstall
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

# Clean up network resources (bridge, iptables rules)
network-clean:
	-sudo ip link del minicontainer0 2>/dev/null
	-sudo iptables -t nat -F PREROUTING 2>/dev/null
	-sudo iptables -t nat -F OUTPUT 2>/dev/null
	-sudo iptables -t nat -F POSTROUTING 2>/dev/null
	@echo "Network resources cleaned"

# Clean everything (build + containers + network)
clean-all: clean network-clean
	sudo rm -rf /var/lib/minicontainer/containers/*
	sudo rm -rf /tmp/minicontainer-overlay-*
	@echo "All resources cleaned"

# Show help
help:
	@echo "MiniContainer Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build         Build the binary"
	@echo "  dev           Build with race detector"
	@echo "  test          Run tests (requires sudo)"
	@echo "  check         Run fmt, vet, build"
	@echo "  clean         Remove build artifacts"
	@echo "  clean-all     Clean build + containers + network"
	@echo "  network-clean Clean bridge and iptables rules"
	@echo "  install       Install to /usr/local/bin"
	@echo "  uninstall     Remove from /usr/local/bin"
	@echo "  help          Show this help"
