# Security Scanning

This document describes the security scanning setup for the GoLikeit project.

## Overview

The project includes comprehensive security scanning to detect vulnerabilities in:
- Source code (static analysis)
- Dependencies (known CVEs)
- Security anti-patterns (SQL injection, hardcoded credentials)

## Tools

### 1. govulncheck

Checks for known vulnerabilities in dependencies using the Go vulnerability database.

```bash
make security-govulncheck
```

**Install:** `go install golang.org/x/vuln/cmd/govulncheck@latest`

### 2. gosec

Security-focused linting that detects:
- SQL injection vulnerabilities
- Hardcoded credentials
- Insecure random number generation
- Path traversal
- Unsafe defer statements

```bash
make security-gosec
```

**Install:** `go install github.com/securego/gosec/v2/cmd/gosec@latest`

### 3. staticcheck

Advanced static analysis that detects:
- Unused code
- Shadowed variables
- Incorrect error handling
- Performance issues
- Security-sensitive patterns

```bash
make security-staticcheck
```

**Install:** `go install honnef.co/go/tools/cmd/staticcheck@latest`

### 4. nancy

Dependency vulnerability scanner using Sonatype's vulnerability database.

```bash
make security-nancy
```

**Install:** `go install github.com/sonatypecommunity/nancy@latest`

## Usage

### Run All Security Scans

```bash
make security
```

This runs govulncheck, gosec, and staticcheck.

### Generate Security Report

```bash
make security-report
```

Creates a comprehensive security report in `reports/security-report.md`.

## CI Integration

Security scans run automatically on:
- Every push to `main` or `develop`
- Every pull request to `main` or `develop`
- Daily at 00:00 UTC (scheduled scan)

## Policy

### Fail-on-High-Severity

The CI pipeline will fail if:
- `gosec` reports HIGH or CRITICAL findings
- `govulncheck` finds vulnerabilities in direct dependencies
- `staticcheck` reports critical issues

### SARIF Reports

Security findings are uploaded as SARIF files to GitHub Security tab.

## Configuration

### Excluding False Positives

To exclude a false positive from gosec:

```go
// #nosec G101
const apiKey = "test-key" // This is a test key, not production
```

To exclude from staticcheck:

```go
//lint:ignore SA1019 This deprecated function is required for backward compatibility
oldFunction()
```

## Maintenance

### Updating Security Tools

Update tools regularly to get the latest vulnerability databases:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/sonatypecommunity/nancy@latest
```

### Vulnerability Response

If a vulnerability is found:

1. Check if it's a direct dependency
2. Update the dependency: `go get -u package@latest`
3. Run tests to ensure compatibility
4. Commit with reference to the CVE

## References

- [Go Vulnerability Database](https://pkg.go.dev/vuln)
- [gosec Rules](https://securego.io/docs/rules.html)
- [staticcheck Checks](https://staticcheck.dev/docs/checks)
- [Nancy Documentation](https://github.com/sonatypecommunity/nancy)
