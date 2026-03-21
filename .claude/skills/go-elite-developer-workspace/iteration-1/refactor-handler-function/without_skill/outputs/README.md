# Refactored Go HTTP Handler

This is a refactored version of a Go HTTP handler that addresses common anti-patterns and follows Go best practices.

## Problems Solved

### 1. Package-level db variable (global state)
**Before:** `var db *sql.DB` at package level

**After:** Database connection is injected through the constructor pattern:
- `NewSQLUserRepository(db *sql.DB)` - Repository receives db via constructor
- `NewUserHandler(service *service.UserService, logger *slog.Logger)` - Handler receives dependencies via constructor
- `main.go` shows how to wire everything together

### 2. No input validation
**Before:** Raw user input was passed directly to database

**After:** Validation in `models/user.go`:
- `Validate()` method checks for required fields
- Name length limits enforced (max 255 characters)
- `Sanitize()` removes leading/trailing whitespace
- Validation errors are defined as package-level variables for easy checking

### 3. Unhandled errors
**Before:** Errors silently ignored with `data, _ := json.Marshal(users)` and `id, _ := result.LastInsertId()`

**After:** All errors are properly handled:
- Repository wraps errors with context using `fmt.Errorf("...: %w", err)`
- Service layer returns meaningful errors
- Handler returns appropriate HTTP status codes (400 for validation, 500 for internal errors)
- JSON encoding errors are logged

### 4. No repository abstraction
**Before:** SQL directly in handler

**After:** Repository pattern in `repository/user_repository.go`:
- `UserRepository` interface defines the contract
- `SQLUserRepository` implements the interface
- Easy to swap implementations (e.g., for testing or different storage backends)

### 5. Handler doing too many things
**Before:** Single handler handled HTTP, business logic, and database access

**After:** Clean separation of concerns:
- **Handler** (`handler/user_handler.go`): HTTP-specific logic (request/response, status codes)
- **Service** (`service/user_service.go`): Business logic (validation, orchestration)
- **Repository** (`repository/user_repository.go`): Data access (SQL queries)
- **Models** (`models/user.go`): Domain types and validation

### 6. No tests
**Before:** No test coverage

**After:** Comprehensive test suite:
- `user_test.go` - Model validation tests
- `user_service_test.go` - Business logic tests with mock repository
- `user_handler_test.go` - HTTP handler tests with mock service

## File Structure

```
.
├── go.mod                      # Module definition
├── main.go                     # Application entry point (wiring)
├── models/
│   ├── user.go                 # User model with validation
│   └── user_test.go            # Model tests
├── repository/
│   └── user_repository.go      # Repository interface and SQL implementation
├── service/
│   ├── user_service.go         # Business logic
│   └── user_service_test.go    # Service tests
└── handler/
    ├── user_handler.go         # HTTP handler
    └── user_handler_test.go    # Handler tests
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./handler/...
go test ./service/...
go test ./models/...
```

## Key Design Decisions

1. **Interface-based design**: Repository uses an interface, allowing easy mocking for tests
2. **Context propagation**: All operations accept `context.Context` for cancellation/timeouts
3. **Structured logging**: Uses `log/slog` for structured, leveled logging
4. **Error wrapping**: Uses `%w` verb to preserve error chains for inspection
5. **Dependency injection**: No globals; all dependencies passed via constructors
6. **Defensive programming**: Input validation at multiple layers (model, service)

## Usage Example

```go
// Setup
db, _ := sql.Open("postgres", dsn)
repo := repository.NewSQLUserRepository(db)
svc := service.NewUserService(repo)
handler := handler.NewUserHandler(svc, logger)

// Register
http.Handle("/users", handler)
http.ListenAndServe(":8080", nil)
```
