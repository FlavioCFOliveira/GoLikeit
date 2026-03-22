# Audit Logging Specification

## Overview

The system shall maintain a simple, comprehensive audit log of all reaction operations. The audit log records every operation performed (ADD, REPLACE, REMOVE) with minimal metadata to provide a complete history of reaction activities.

## Functional Requirements

### Requirement 1: Mandatory Audit Logging with Configurable Persistence

**Description:** The system shall implement audit logging as an architectural mandatory component. Every reaction operation generates an audit event. The audit system defaults to NullAuditor when no persistence is configured, but the audit logging mechanism is always present and cannot be disabled.

**Requirements:**
- **Architectural Mandate:** Audit logging is a core system component that is always active
  - Every reaction operation (ADD, REPLACE, REMOVE) generates an audit event
  - The audit system cannot be disabled or removed from the module
  - This satisfies security-first policy requiring comprehensive audit trails
- **NullAuditor (Default Fallback):** A no-op implementation used when no persistence is configured
  - NullAuditor accepts audit entries but performs no persistence
  - NullAuditor has zero overhead and no external dependencies
  - NullAuditor is the **fallback**, not a replacement for audit logging
  - Suitable for development, testing, or when external audit storage is unavailable
- **Persistent Auditor:** Full audit logging implementation that persists to configured storage
  - Automatically activated when audit storage is configured
  - Every reaction operation is recorded to persistent storage
  - Audit log entries are append-only and immutable
- **Implementation Selection:** The system automatically selects the appropriate implementation
  - If audit storage is configured: use Persistent Auditor
  - If no audit storage is configured: use NullAuditor (fallback)

**Behavior:**
- By default, if no audit storage is configured, NullAuditor is used (audit events accepted but not persisted)
- Consuming application configures persistent auditing by providing audit storage configuration
- Failed operations are logged with error indication (when using persistent auditor)
- Audit logging occurs as part of the transaction (atomic with the operation) when persistent
- The audit system is always initialized and cannot be "turned off"

**Rationale:**
- Security-first: Audit logging is architecturally mandatory, satisfying compliance requirements
- Flexibility: Applications can run without external audit storage (using NullAuditor fallback)
- Zero-overhead option: NullAuditor provides fallback with no performance impact
- No breaking changes: Existing applications without audit configuration continue to work
- Gradual adoption: Applications can add audit persistence without code changes, only configuration

### Requirement 2: Log Entry Content

**Description:** Each audit log entry shall contain the minimal required information to identify the operation.

**Required Fields:**
- **id:** Unique identifier for the audit entry
- **operation:** The operation type (ADD, REPLACE, REMOVE)
- **user_id:** The user who performed the operation
- **entity_type:** The type of entity affected
- **entity_id:** The identifier of the entity affected
- **reaction_type:** The resulting reaction type (e.g., "LIKE", "LOVE")
- **previous_reaction:** The reaction type before the operation (empty if no previous reaction)
- **timestamp:** The exact date and time when the operation occurred

**Operation Types:**
- **ADD:** User added a new reaction where none existed
- **REPLACE:** User replaced an existing reaction with a different type
- **REMOVE:** User removed their reaction

### Requirement 3: Timestamp Precision

**Description:** Audit log timestamps shall capture the exact moment of the operation.

**Requirements:**
- Timestamps shall be recorded with millisecond precision
- All timestamps shall be stored in UTC
- The timestamp represents the moment the operation was executed
- Clock skew shall be handled gracefully (monotonic clock where available)

### Requirement 4: Audit Log Storage

**Description:** Audit log entries shall be stored in a separate table/collection from reaction data.

**Requirements:**
- Audit log table shall be separate from reaction data table
- Audit log entries shall be stored in the same transaction as the operation (when same database)
- Failed operations may be logged with error status

**Retention:**
- No automatic purge or retention limit by default
- Consuming application may implement purge policies
- Audit log may grow large and should be considered in capacity planning

### Requirement 5: Query Capabilities

**Description:** The system shall provide query capabilities for the audit log.

**Required Queries:**
- Retrieve audit entries by user_id (chronological order)
- Retrieve audit entries by entity (type + id) (chronological order)
- Retrieve audit entries by operation type
- Retrieve audit entries by reaction type
- Retrieve audit entries by date range
- Support pagination for large result sets

**Performance:**
- Queries shall use appropriate indexes for efficient retrieval
- Default sort order: newest first (timestamp descending)

### Requirement 6: Immutability

**Description:** Audit log entries shall be immutable once created.

**Requirements:**
- Audit log entries cannot be modified after creation
- Audit log entries cannot be deleted through the API
- Updates to reactions create new audit entries, do not modify existing ones

**Rationale:**
- Provides complete historical record
- Prevents tampering with activity history
- Supports compliance and debugging requirements

### Requirement 7: Audit Operations Restriction

**Description:** The audit package shall support only insert and get operations; delete operations are strictly prohibited.

**Requirements:**
- **Insert Only:** Audit log entries can only be created, never modified or deleted
- **Get Only:** Audit log entries can be queried and retrieved, but no mutation operations are exposed
- **No Delete API:** No method shall exist to remove audit entries programmatically
- **No Update API:** No method shall exist to modify existing audit entries

**Allowed Operations:**
- Insert audit entry (atomic with reaction operation)
- Get audit entry by ID
- Query audit entries (by user, entity, date range, operation type, reaction type)
- Paginate through audit entries

**Rationale:**
- Ensures audit trail integrity and non-repudiation
- Prevents accidental or malicious tampering with history
- Supports compliance requirements (e.g., GDPR audit trails, financial regulations)

### Requirement 8: Independent Storage Layer

**Description:** The audit package shall be capable of operating on a separate data layer from where reactions are persisted.

**Requirements:**
- Audit storage configuration shall be independent from reaction storage configuration
- The system shall support different database instances for audit and reaction data
- Audit operations shall not depend on reaction storage availability
- Audit storage failures shall not impact reaction operations (and vice versa)
- Cross-database transactions are not required; eventual consistency is acceptable for audit entries

**Audit Failure Behavior:**
- If audit storage fails during a reaction operation, the reaction operation **continues and succeeds**
- The reaction data is persisted to reaction storage regardless of audit failure
- Audit failure is logged as an error (via module logger) but does not block the operation
- Failed audit entries may be queued for retry (implementation-dependent)
- Eventual consistency: audit entry may appear after the reaction is already visible
- Priority: Availability of reaction functionality takes precedence over audit persistence

**Configuration Options:**
- Separate connection strings for audit and reaction storage
- Independent connection pool settings for each storage
- Optional: different database types (e.g., PostgreSQL for reactions, ClickHouse for audit)

**Rationale:**
- Allows audit data to be stored in specialized systems optimized for write-heavy workloads
- Enables different retention and backup policies for audit vs. operational data
- Prevents audit storage issues from impacting core reaction functionality
- Supports regulatory requirements for isolated audit storage

## Constraints and Limitations

1. **Storage Growth:** The audit log grows monotonically and may consume significant storage over time. The consuming application is responsible for retention policies.

2. **No Filtering at Source:** All operations are logged; selective filtering is not supported.

3. **Independent Storage:** Audit log may be stored in a separate database from reaction data; cross-database consistency is eventual, not transactional.

4. **No Real-Time Streaming:** Audit log is query-based; real-time streaming to external systems is not provided (may be added by consuming application).

5. **No Delete/Update Operations:** The audit package explicitly does not provide delete or update operations; audit entries are immutable and append-only.

6. **Source of Historical Truth:** Since reaction_management.md implements hard delete (RemoveReaction permanently deletes), the audit log is the sole source of historical data for removed reactions. Applications requiring "who reacted before" or "reaction history" functionality must query the audit log.

7. **Time-Series Analytics:** The audit log contains timestamps for all operations and can be used for time-series analysis (e.g., reactions over time, engagement trends). However, dedicated time-series analytics features (aggregations, rollups, trend analysis) are not provided by this module. Applications requiring such analytics should process the audit log or use dedicated analytics systems.

## Relationships with Other Functional Blocks

- **[reaction_management.md](reaction_management.md):** Defines the operations that generate audit log entries
- **[data_persistence.md](data_persistence.md):** Defines the storage mechanism for audit log
- **[user_interactions.md](user_interactions.md):** Defines user-based audit log queries
- **[entity_association.md](entity_association.md):** Defines entity-based audit log queries

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version of audit logging specification |
| 2026-03-21 | Update | Added Requirement 7 (Audit Operations Restriction) and Requirement 8 (Independent Storage Layer) |
| 2026-03-21 | Update | Modified Requirement 1 to Configurable Audit Logging with NullAuditor as default |
| 2026-03-21 | Clarification | Requirement 1 clarified: Audit logging is architecturally mandatory; NullAuditor is the fallback when no persistence is configured |
| 2026-03-22 | Update | Updated operation types from LIKE/UNLIKE/DISLIKE/UNDISLIKE to ADD/REPLACE/REMOVE |

## Acceptance Criteria

1. **AC1:** Every ADD operation creates an audit log entry
2. **AC2:** Every REPLACE operation creates an audit log entry
3. **AC3:** Every REMOVE operation creates an audit log entry
4. **AC4:** Audit log entries include all required fields (id, operation, user_id, entity_type, entity_id, reaction_type, previous_reaction, timestamp)
5. **AC5:** Audit log timestamps are recorded in UTC with millisecond precision
6. **AC6:** Audit log entries are immutable and cannot be modified or deleted
7. **AC7:** Audit log entries can be queried by user_id with pagination
8. **AC8:** Audit log entries can be queried by entity (type + id) with pagination
9. **AC9:** Audit log entries can be queried by reaction type
10. **AC10:** Audit log queries support date range filtering
11. **AC11:** Audit log entries are created atomically with the corresponding reaction operation (when same storage)
12. **AC12:** The audit package exposes only Insert and Get operations; no Delete or Update methods exist
13. **AC13:** Audit storage can be configured independently from reaction storage with separate connection parameters
14. **AC14:** Audit storage failures do not impact reaction operations (decoupled storage)
15. **AC15:** NullAuditor is available as default no-op implementation
16. **AC16:** Consuming application can configure persistent auditing explicitly
17. **AC17:** Switching between NullAuditor and persistent auditor requires no code changes
