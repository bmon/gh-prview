.PHONY: build test clean

# Default target
all: build

# Build the executable
build:
	go build -o gh-prview ./cmd

# Run tests
test:
	go test -v ./...
