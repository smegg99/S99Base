# justfile

# List available commands.
default:
    @just --list

# Check all packages without rewriting source files.
check: check-fmt vet test

# Format all Go files.
fmt:
    gofmt -l -w .

# Fail on unformatted Go files without changing them.
check-fmt:
    @files=$(gofmt -l .); test -z "$files" || { printf '%s\n' "$files"; exit 1; }

# Run go vet.
vet:
    go vet ./...

# Run all tests.
test:
    go test -race ./...
