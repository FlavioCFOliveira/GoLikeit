# Audit Logging Specification

## Overview

The system shall maintain a simple, comprehensive audit log of all reaction operations. The audit log records every operation performed (LIKE, UNLIKE, DISLIKE, UNDISLIKE) with minimal metadata to provide a complete history of reaction activities.

## Functional Requirements

### Requirement 1: Configurable Audit Logging

**Description:** The system shall support configurable audit logging with a no-op implementation as default.

**Requirements:**
- Audit logging is configurable; consuming application chooses the implementation
- **NullAuditor:** A no-op (null object) implementation is provided as default
  - NullAuditor accepts audit entries but performs no persistence
  - NullAuditor has zero overhead and no external dependencies
  - NullAuditor is suitable when audit logging is not required
- **Persistent Auditor:** Full audit logging implementation persists to configured storage
  - Every reaction operation is recorded
  - Audit log entries are append-only and immutable
- Audit implementation is selected during module initialization

**Behavior:**
- By default, NullAuditor is used (no audit output)
- Consuming application must explicitly configure persistent auditing if desired
- Failed operations may still be logged with error indication (when using persistent auditor)
- Audit logging occurs as part of the transaction (atomic with the operation) when persistent

**Rationale:**
- Zero-overhead default for applications that do not require auditing
- No breaking changes when upgrading from versions without auditing
- Consuming application explicitly opts into audit functionality

### Requirement 2: Log Entry Content

**Description:** Each audit log entry shall contain the minimal required information to identify the operation.

**Required Fields:**
- **id:** Unique identifier for the audit entry
- **operation:** The operation type (LIKE, UNLIKE, DISLIKE, UNDISLIKE, CONVERSION_LIKE, CONVERSION_DISLIKE)
- **user_id:** The user who performed the operation
- **entity_type:** The type of entity affected
- **entity_id:** The identifier of the entity affected
- **reaction_type:** The resulting reaction state (LIKE, DISLIKE, NONE for removals)
- **previous_reaction:** The reaction state before the operation (LIKE, DISLIKE, NONE)
- **timestamp:** The exact date and time when the operation occurred

**Operation Types:**
- **LIKE:** User created a new LIKE
- **UNLIKE:** User removed an existing LIKE
- **DISLIKE:** User created a new DISLIKE
- **UNDISLIKE:** User removed an existing DISLIKE
- **CONVERSION_LIKE:** LIKE replaced an existing DISLIKE
- **CONVERSION_DISLIKE:** DISLIKE replaced an existing LIKE

### Requirement 3: Timestamp Precision

**Description:** Audit log timestamps shall capture the exact moment of the operation.

**Requirements:**
- Timestamps shall be recorded with millisecond precision
- All timestamps shall be stored in UTC
- The timestamp represents the moment the operation was executed
- Clock skew shall be handled gracefully (monotonic clock where available)

### Requirement 4: Audit Log Storage

**Description:** Audit log entries shall be stored in the same database as reaction data.

**Requirements:**
- Audit log table shall be separate from reaction data table
- Audit log entries shall be stored in the same transaction as the operation
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
- Query audit entries (by user, entity, date range, operation type)
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

## Acceptance Criteria

1. **AC1:** Every LIKE operation creates an audit log entry
2. **AC2:** Every UNLIKE operation creates an audit log entry
3. **AC3:** Every DISLIKE operation creates an audit log entry
4. **AC4:** Every UNDISLIKE operation creates an audit log entry
5. **AC5:** Conversion operations (LIKE replacing DISLIKE, DISLIKE replacing LIKE) create audit log entries with appropriate operation type
6. **AC6:** Audit log entries include all required fields (id, operation, user_id, entity_type, entity_id, reaction_type, previous_reaction, timestamp)
7. **AC7:** Audit log timestamps are recorded in UTC with millisecond precision
8. **AC8:** Audit log entries are immutable and cannot be modified or deleted
9. **AC9:** Audit log entries can be queried by user_id with pagination
10. **AC10:** Audit log entries can be queried by entity (type + id) with pagination
11. **AC11:** Audit log queries support date range filtering
12. **AC12:** Audit log entries are created atomically with the corresponding reaction operation
13. **AC13:** The audit package exposes only Insert and Get operations; no Delete or Update methods exist
14. **AC14:** Audit storage can be configured independently from reaction storage with separate connection parameters
15. **AC15:** Audit storage failures do not impact reaction operations (decoupled storage)
16. **AC16:** NullAuditor is available as default no-op implementation
17. **AC17:** Consuming application can configure persistent auditing explicitly
18. **AC18:** Switching between NullAuditor and persistent auditor requires no code changes
