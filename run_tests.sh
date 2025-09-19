#!/bin/bash

# Load environment variables from .env file
if [ -f .env ]; then
    echo "Loading environment variables from .env..."
    set -a
    source .env
    set +a
    echo "Environment variables loaded successfully"
else
    echo "Warning: .env file not found"
fi

# Run tests with loaded environment variables
echo "Running tests..."
go test ./... "$@"