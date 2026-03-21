# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands
- **Build**: `go build ./...`
- **Test**: `go test ./...`
- **Single Test**: `go test -v -run <TestName> ./...`
- **Lint**: `golangci-lint run` (if available) or `go vet ./...`

## Architecture
This project is a Go module designed to add "Like" functionality to applications. As a library/module, it focuses on providing a clean API for managing likes on various entities.

- **Storage**: Likely to support multiple storage backends (SQL, Redis, etc.) via interfaces.
- **API**: Designed to be integrated into other Go services.
