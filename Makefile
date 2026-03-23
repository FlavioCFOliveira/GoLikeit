# GoLikeit Makefile
# Provides standardized build automation commands

.PHONY: help build test test-unit test-integration test-e2e lint coverage clean fmt vet security

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod

# Binary name
BINARY_NAME := golikeit

# Build flags
BUILD_FLAGS := -v
TEST_FLAGS := -v -race
COVERAGE_FLAGS := -race -coverprofile=coverage.out -covermode=atomic

## help: Show this help message
help:
	@echo "Available targets:"
	@awk '/^##/ { sub(/^## /, ""); print "" } /^[a-zA-Z_-]+:/ { sub(/:.*$$/, ""); printf "  %-20s\n", $$0 }' $(MAKEFILE_LIST)

## build: Compile all packages
build:
	$(GOBUILD) $(BUILD_FLAGS) ./...

## test: Run all tests
 test:
	$(GOTEST) $(TEST_FLAGS) ./...

## test-unit: Run unit tests only (excludes integration and e2e)
test-unit:
	$(GOTEST) $(TEST_FLAGS) -short ./...

## test-integration: Run integration tests
test-integration:
	$(GOTEST) $(TEST_FLAGS) -tags=integration ./...

## test-e2e: Run end-to-end tests
test-e2e:
	$(GOTEST) $(TEST_FLAGS) -tags=e2e ./...

## test-storage: Run storage-specific tests
test-storage:
	$(GOTEST) $(TEST_FLAGS) ./storage/...

## lint: Run golangci-lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

## vet: Run go vet
vet:
	$(GOVET) ./...

## fmt: Format Go code
fmt:
	$(GOFMT) -w .

## coverage: Generate coverage report
coverage:
	$(GOTEST) $(COVERAGE_FLAGS) ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## coverage-report: Display coverage summary
coverage-report: coverage
	$(GOCMD) tool cover -func=coverage.out | tail -1

## security: Run security scanning tools
security: security-govulncheck security-gosec security-staticcheck
	@echo "All security checks completed"

## security-govulncheck: Check for known vulnerabilities in dependencies
security-govulncheck:
	@if command -v govulncheck >/dev/null 2>&1; then \
		echo "Running govulncheck..."; \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

## security-gosec: Run security-focused linting
security-gosec:
	@if command -v gosec >/dev/null 2>&1; then \
		echo "Running gosec..."; \
		gosec -fmt text -severity high -confidence medium ./...; \
	else \
		echo "gosec not installed. Run: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi

## security-staticcheck: Run advanced static analysis
security-staticcheck:
	@if command -v staticcheck >/dev/null 2>&1; then \
		echo "Running staticcheck..."; \
		staticcheck -f text ./...; \
	else \
		echo "staticcheck not installed. Run: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
		exit 1; \
	fi

## security-nancy: Check dependencies for vulnerabilities using nancy
security-nancy:
	@if command -v nancy >/dev/null 2>&1; then \
		echo "Running nancy..."; \
		$(GOMOD) tidy; \
		$(GOCMD) list -json -deps ./... | nancy sleuth --exclude-vulnerability-cache || true; \
	else \
		echo "nancy not installed. Run: go install github.com/sonatypecommunity/nancy@latest"; \
		exit 1; \
	fi

## security-report: Generate comprehensive security report
security-report:
	@mkdir -p reports
	@echo "Generating security report..."
	@echo "# Security Scan Report" > reports/security-report.md
	@echo "Generated: $$(date)" >> reports/security-report.md
	@echo "" >> reports/security-report.md
	@echo "## govulncheck" >> reports/security-report.md
	@echo '```' >> reports/security-report.md
	@-$(MAKE) security-govulncheck 2>&1 | tee -a reports/security-report.md || true
	@echo '```' >> reports/security-report.md
	@echo "" >> reports/security-report.md
	@echo "## gosec" >> reports/security-report.md
	@echo '```' >> reports/security-report.md
	@-$(MAKE) security-gosec 2>&1 | tee -a reports/security-report.md || true
	@echo '```' >> reports/security-report.md
	@echo "" >> reports/security-report.md
	@echo "## staticcheck" >> reports/security-report.md
	@echo '```' >> reports/security-report.md
	@-$(MAKE) security-staticcheck 2>&1 | tee -a reports/security-report.md || true
	@echo '```' >> reports/security-report.md
	@echo "Security report generated: reports/security-report.md"

## clean: Remove build artifacts and test files
clean:
	rm -f coverage.out coverage.html
	rm -rf reports/
	$(GOCMD) clean -testcache

## deps: Download and verify dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

## deps-update: Update dependencies to latest versions
deps-update:
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

## bench: Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem ./...

## ci: Run all CI checks (build, test, lint, security)
ci: fmt vet build test lint security
	@echo "All CI checks passed"

## fuzz: Run fuzz tests for critical functions
fuzz:
	@echo "Running fuzz tests..."
	$(GOTEST) -tags=gofuzz -run=Fuzz -v ./validation/... ./golikeit/... ./business/...

## fuzz-short: Run fuzz tests with limited iterations
fuzz-short:
	@echo "Running short fuzz tests (30 seconds per test)..."
	$(GOTEST) -tags=gofuzz -fuzztime=30s -run=Fuzz ./validation/... ./golikeit/... ./business/...

## fuzz-validation: Run validation fuzz tests only
fuzz-validation:
	$(GOTEST) -tags=gofuzz -run=Fuzz -v ./validation/...

## fuzz-domain: Run domain fuzz tests only
fuzz-domain:
	$(GOTEST) -tags=gofuzz -run=Fuzz -v ./golikeit/...

## fuzz-business: Run business fuzz tests only
fuzz-business:
	$(GOTEST) -tags=gofuzz -run=Fuzz -v ./business/...
