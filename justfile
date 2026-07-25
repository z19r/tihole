# tihole — task runner
# Run `just` or `just --list` to see recipes.

bin := "tihole"
pkg := "./cmd/tihole"
out := "bin/tihole"

# Show available recipes
default:
    @just --list

# Build the binary into ./bin
build:
    go build -o {{ out }} {{ pkg }}

# Run the TUI
run *args:
    go run {{ pkg }} {{ args }}

# Install to $GOBIN / $GOPATH/bin
install:
    go install {{ pkg }}

# Run all tests
test *args:
    go test ./... {{ args }}

# Run tests with coverage summary (house standard: 80%+)
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1

# Open the HTML coverage report
cover-html: cover
    go tool cover -html=coverage.out

# Format all Go source (gofmt + golines at 80 cols)
fmt:
    gofmt -w .
    PATH="$(go env GOPATH)/bin:$PATH" golines \
        --base-formatter=gofmt --max-len=80 \
        --shorten-comments -w . \
        || go run github.com/segmentio/golines@latest \
        --base-formatter=gofmt --max-len=80 \
        --shorten-comments -w .

# Verify formatting (fails if anything is unformatted)
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    bad="$(gofmt -l .)"
    if [[ -n "$bad" ]]; then
        echo "$bad"
        exit 1
    fi
    export PATH="$(go env GOPATH)/bin:$PATH"
    if ! command -v golines >/dev/null; then
        go install github.com/segmentio/golines@latest
    fi
    bad="$(golines --base-formatter=gofmt --max-len=80 \
        --shorten-comments -l .)"
    if [[ -n "$bad" ]]; then
        echo "$bad"
        exit 1
    fi

# go vet
vet:
    go vet ./...

# Full lint gate: formatting + vet
lint: fmt-check vet

# Tidy modules
tidy:
    go mod tidy

# Everything CI should enforce
check: lint test

# Remove build + coverage artifacts
clean:
    rm -rf bin coverage.out
