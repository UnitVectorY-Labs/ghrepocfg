
# Commands for ghrepocfg
default:
  @just --list
# Build ghrepocfg with Go
build:
  go build ./...

# Run tests for ghrepocfg with Go
test:
  go clean -testcache
  go test ./...