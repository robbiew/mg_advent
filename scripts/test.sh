#!/bin/bash

# Test script for Mistigris Advent Calendar
# Runs all tests and generates coverage reports

set -e

echo "Running tests for Mistigris Advent Calendar..."

# Run unit tests with coverage
echo "Running unit tests..."
go test -v -race -coverprofile=coverage.out ./...

# Generate coverage report
echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out

# Run integration tests if they exist
if [ -d "tests/integration" ]; then
    echo "Running integration tests..."
    go test -v ./tests/integration/...
fi

# Run benchmarks
echo "Running benchmarks..."
go test -bench=. -benchmem ./...

echo "Tests complete!"
echo "Coverage report: coverage.html"
echo "Coverage summary:"
go tool cover -func=coverage.out | tail -1