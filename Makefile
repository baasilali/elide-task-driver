# Makefile for Elide Task Driver

.PHONY: build test clean fmt vet install

# Binary name
BINARY_NAME=elide-task-driver

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Build directory
BUILD_DIR=build
PLUGIN_DIR=$(BUILD_DIR)/plugins

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(PLUGIN_DIR)
	$(GOBUILD) -o $(PLUGIN_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(PLUGIN_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short ./...

fmt:
	@echo "Formatting code..."
	$(GOFMT) -w .

vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

lint: fmt vet
	@echo "Linting complete"

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

install: build
	@echo "Installing plugin..."
	@mkdir -p /tmp/nomad-plugins
	cp $(PLUGIN_DIR)/$(BINARY_NAME) /tmp/nomad-plugins/
	@echo "Plugin installed to /tmp/nomad-plugins/$(BINARY_NAME)"

# Generate proto code (when proto definitions are available)
proto:
	@echo "Generating proto code..."
	@echo "TODO: Implement once Elide proto definitions are available"
	@echo "This will use protoc or buf to generate Go stubs from proto files"

# Development helpers
dev-setup: deps
	@echo "Development environment setup complete"

all: clean deps fmt vet test build

