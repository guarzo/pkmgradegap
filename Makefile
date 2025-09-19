# Pokemon Grade Gap Analyzer - Makefile

.PHONY: help build test coverage lint fmt clean install web

# Default target
help:
	@echo "Pokemon Grade Gap Analyzer"
	@echo ""
	@echo "Essential targets:"
	@echo "  build         Build the CLI binary"
	@echo "  web           Start web server with .env loaded (production mode)"
	@echo "  test          Run all tests with mocks"
	@echo "  coverage      Run tests with coverage report"
	@echo "  lint          Run linting and formatting"
	@echo "  clean         Clean build artifacts"
	@echo "  install       Install dependencies"
	@echo ""
	@echo "Note: Use '/usr/bin/make web' if 'make' command conflicts with shell function"

# Build
build:
	@echo "Building CLI..."
	go build -o pkmgradegap ./cmd/pkmgradegap

# Web server
web:
	@echo "Starting web server with .env loaded..."
	@if [ -f .env ]; then \
		bash -c 'set -a && source .env && set +a && go run ./cmd/pkmgradegap --web --with-pop-api --port $${PORT:-8081}'; \
	else \
		echo "Warning: .env file not found"; \
		go run ./cmd/pkmgradegap --web --port 8080; \
	fi

# Testing
test:
	@echo "Running tests..."
	TEST_MODE=true GAMESTOP_MOCK=true SALES_MOCK=true POPULATION_MOCK=true go test -v ./...

coverage:
	@echo "Running tests with coverage..."
	TEST_MODE=true GAMESTOP_MOCK=true SALES_MOCK=true POPULATION_MOCK=true \
		go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Code quality
lint:
	@echo "Formatting and linting..."
	go fmt ./...
	go vet ./...
	golangci-lint run

fmt:
	go fmt ./...

# Maintenance
clean:
	@echo "Cleaning..."
	rm -f pkmgradegap coverage.out coverage.html
	go clean

install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# CI target
ci: install lint test coverage build
	@echo "CI complete"