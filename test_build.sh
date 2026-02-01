#!/bin/bash
cd "$(dirname "$0")"
echo "Testing Go compilation..."
go mod tidy
go build ./...
echo "Build result: $?"