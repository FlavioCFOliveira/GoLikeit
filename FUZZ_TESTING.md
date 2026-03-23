# Fuzz Testing

This document describes the fuzz testing setup for the GoLikeit project.

## Overview

Fuzz testing is a technique for finding security vulnerabilities and bugs by automatically generating random inputs. The project includes fuzz tests for critical functions to discover edge cases and potential crashes.

## Running Fuzz Tests

### Run All Fuzz Tests

```bash
make fuzz
```

### Run Short Fuzz Tests (30 seconds each)

```bash
make fuzz-short
```

### Run Specific Fuzz Tests

```bash
# Validation fuzz tests
make fuzz-validation

# Domain fuzz tests
g make fuzz-domain

# Business fuzz tests
make fuzz-business
```

### Run Individual Fuzz Test

```bash
go test -tags=gofuzz -run=FuzzValidateReactionID ./validation/...
```

## Fuzz Test Coverage

### Validation Package (`validation/fuzz_test.go`)

Tests for input validation functions:
- `FuzzValidateReactionID` - Tests reaction ID validation
- `FuzzValidateEntityID` - Tests entity ID validation
- `FuzzValidateUserID` - Tests user ID validation
- `FuzzValidateEntityType` - Tests entity type validation
- `FuzzValidateReactionType` - Tests reaction type validation
- `FuzzValidateTimestamp` - Tests timestamp parsing
- `FuzzValidatePagination` - Tests pagination parameter validation
- `FuzzValidateConfig` - Tests configuration validation

### Domain Package (`golikeit/fuzz_test.go`)

Tests for domain types and serialization:
- `FuzzEntityTargetString` - Tests EntityTarget methods
- `FuzzReactionTypeValidation` - Tests reaction type format validation
- `FuzzUserReactionSerialization` - Tests JSON marshaling/unmarshaling
- `FuzzEntityCountsSerialization` - Tests EntityCounts JSON handling
- `FuzzPaginationValidation` - Tests pagination validation
- `FuzzPaginatedResultSerialization` - Tests paginated result JSON
- `FuzzTimestampParsing` - Tests timestamp parsing
- `FuzzConfigValidation` - Tests configuration validation

### Business Package (`business/fuzz_test.go`)

Tests for business logic:
- `FuzzConfigValidation` - Tests business configuration validation
- `FuzzReactionTypeValidation` - Tests reaction type against configured types
- `FuzzServiceInputValidation` - Tests service method input validation
- `FuzzPaginationValidation` - Tests pagination validation for queries
- `FuzzBulkOperationInputs` - Tests bulk operation input validation
- `FuzzConfigJSONSerialization` - Tests Config JSON handling

## Seed Corpus

Each fuzz test includes a seed corpus with:
- Valid inputs
- Edge cases (empty strings, maximum lengths)
- Invalid inputs (special characters, null bytes, newlines)
- Boundary conditions

## Target: 1M Iterations

To run fuzz tests with the recommended 1M iterations:

```bash
go test -tags=gofuzz -fuzz=FuzzValidateReactionID -fuzztime=10m ./validation/...
```

## Continuous Fuzzing

For continuous fuzzing, consider using:
- [OSS-Fuzz](https://github.com/google/oss-fuzz)
- [ClusterFuzz](https://github.com/google/clusterfuzz)
- [Fuzzit](https://fuzzit.dev/)

## Crash Reproduction

If a fuzz test finds a crash, it will save the input in:
- `testdata/fuzz/`

To reproduce:

```bash
go test -tags=gofuzz -run=FuzzValidateReactionID ./validation/... -v
```

## Best Practices

1. **Seed with diverse inputs** - Include valid and invalid cases
2. **Test boundaries** - Empty strings, max lengths, null bytes
3. **Check for panics** - Fuzz tests should not panic
4. **Verify serialization** - Test marshal/unmarshal round-trips
5. **Monitor coverage** - Use `-cover` to see fuzzing coverage

## References

- [Go Fuzzing Documentation](https://go.dev/doc/security/fuzz/)
- [Fuzzing Tutorial](https://go.dev/doc/tutorial/fuzz)
- [Security Best Practices](https://go.dev/doc/security)
