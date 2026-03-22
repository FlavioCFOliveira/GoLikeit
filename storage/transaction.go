// Package storage defines transaction support for atomic storage operations.
package storage

import (
	"context"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
)

// Transaction defines the interface for database transactions.
// Transactions allow multiple operations to be executed atomically.
//
// Implementations must ensure ACID properties:
//   - Atomicity: All operations succeed or none do
//   - Consistency: Database constraints are maintained
//   - Isolation: Concurrent transactions don't interfere
//   - Durability: Committed changes persist
//
// Example usage:
//
//	tx, err := repo.BeginTx(ctx)
//	if err != nil {
//	    return err
//	}
//	defer tx.Rollback() // Safe to call even after Commit
//
//	// Perform operations
//	if _, err := tx.AddReaction(ctx, userID, target, reactionType); err != nil {
//	    return err // Automatically rolled back by defer
//	}
//
//	return tx.Commit()
type Transaction interface {
	// AddReaction adds a reaction within the transaction.
	// See Repository.AddReaction for details.
	AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error)

	// RemoveReaction removes a reaction within the transaction.
	// See Repository.RemoveReaction for details.
	RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error

	// GetUserReaction retrieves a reaction within the transaction.
	// See Repository.GetUserReaction for details.
	GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error)

	// Commit commits the transaction, making all changes permanent.
	// Returns an error if the commit fails.
	Commit() error

	// Rollback aborts the transaction, discarding all changes.
	// Safe to call multiple times and after Commit (no-op).
	Rollback() error
}

// TransactionalRepository extends Repository with transaction support.
// Not all storage backends support transactions (e.g., Redis, Cassandra).
type TransactionalRepository interface {
	Repository

	// BeginTx starts a new transaction.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//
	// Returns:
	//   - tx: The transaction handle
	//   - error: nil on success, or ErrStorageUnavailable if the backend is unavailable
	BeginTx(ctx context.Context) (Transaction, error)

	// BeginTxWithOptions starts a new transaction with options.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - opts: Transaction options (isolation level, read-only, etc.)
	//
	// Returns:
	//   - tx: The transaction handle
	//   - error: nil on success, or ErrStorageUnavailable if the backend is unavailable
	BeginTxWithOptions(ctx context.Context, opts TxOptions) (Transaction, error)
}

// TxOptions defines options for starting a transaction.
type TxOptions struct {
	// IsolationLevel specifies the transaction isolation level.
	// Supported levels vary by backend.
	IsolationLevel IsolationLevel

	// ReadOnly indicates the transaction should be read-only.
	// Read-only transactions may have better performance.
	ReadOnly bool
}

// IsolationLevel defines transaction isolation levels.
type IsolationLevel int

const (
	// LevelDefault uses the default isolation level for the backend.
	LevelDefault IsolationLevel = iota

	// LevelReadUncommitted allows dirty reads.
	LevelReadUncommitted

	// LevelReadCommitted prevents dirty reads.
	LevelReadCommitted

	// LevelWriteCommitted is used by some backends (e.g., Cassandra).
	LevelWriteCommitted

	// LevelSnapshot reads from a consistent snapshot.
	LevelSnapshot

	// LevelRepeatableRead prevents non-repeatable reads.
	LevelRepeatableRead

	// LevelSerializable provides full serializability.
	LevelSerializable

	// LevelLinearizable provides linearizable consistency.
	LevelLinearizable
)

// String returns the string representation of the isolation level.
func (il IsolationLevel) String() string {
	switch il {
	case LevelDefault:
		return "DEFAULT"
	case LevelReadUncommitted:
		return "READ_UNCOMMITTED"
	case LevelReadCommitted:
		return "READ_COMMITTED"
	case LevelWriteCommitted:
		return "WRITE_COMMITTED"
	case LevelSnapshot:
		return "SNAPSHOT"
	case LevelRepeatableRead:
		return "REPEATABLE_READ"
	case LevelSerializable:
		return "SERIALIZABLE"
	case LevelLinearizable:
		return "LINEARIZABLE"
	default:
		return "UNKNOWN"
	}
}

// IsTransactional reports whether the repository supports transactions.
func IsTransactional(repo Repository) bool {
	_, ok := repo.(TransactionalRepository)
	return ok
}

// AsTransactional attempts to cast a Repository to TransactionalRepository.
// Returns nil and false if the repository does not support transactions.
func AsTransactional(repo Repository) (TransactionalRepository, bool) {
	tr, ok := repo.(TransactionalRepository)
	return tr, ok
}
