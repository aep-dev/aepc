#!/usr/bin/env bash
set -e

if ! which buf-plugin-aep > /dev/null; then
    go install github.com/aep-dev/api-linter/cmd/buf-plugin-aep@latest
fi

echo "Running API linter..."
buf lint --path example/bookstore/v1/bookstore.proto
echo "passed!"