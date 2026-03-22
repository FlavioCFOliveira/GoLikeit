# Audit Logging Specification

## Overview

The system maintains a comprehensive audit log of all reaction operations. Audit logging is architecturally mandatory - the audit system is always present.

## Functional Requirements

### Requirement 1: Mandatory Audit Logging

Audit logging is a core system component.

- Every reaction operation generates an audit event
- NullAuditor (no-op) is used when no persistence is configured
- Persistent auditing is activated by configuring audit storage
- The audit system cannot be disabled

### Requirement 2: Log Entry Content

Each audit log entry contains required information.

**Required Fields:**
- **id:** Unique identifier for the audit entry
- **operation:** Operation type (ADD, REPLACE, REMOVE)
- **user_id:** User who performed the operation
- **entity_type:** Type of entity affected
- **entity_id:** Identifier of the entity affected
- **reaction_type:** Resulting reaction type
- **previous_reaction:** Reaction type before operation (empty if none)
- **timestamp:** When the operation occurred

### Requirement 3: Timestamp Precision

Timestamps capture the exact moment of the operation.

- Millisecond precision
- Stored in UTC
- Represents moment the operation was executed

### Requirement 4: Audit Log Storage

Audit log entries are stored separately from reaction data.

- Audit table is separate from reaction data table
- Entries stored atomically with operation (when same database)
- No automatic purge by default

### Requirement 5: Query Capabilities

The system provides query capabilities.

- Retrieve by user_id
- Retrieve by entity (type + id)
- Retrieve by operation type
- Retrieve by reaction type
- Retrieve by date range
- Support pagination

### Requirement 6: Immutability

Audit log entries are immutable.

- Cannot be modified after creation
- Cannot be deleted through the API
- Updates create new entries

### Requirement 7: Independent Storage and Failure Handling

The audit system operates with complete independence from the core reaction storage to ensure maximum availability of the primary service.

**Failure Behavior:**
1.  **Non-Blocking Operations**: Audit logging MUST NOT block or impact the performance of the primary reaction operations.
2.  **Fire-and-Forget**: If the audit storage is unavailable or an audit write fails, the audit entry is discarded. The primary reaction operation remains valid and persisted.
3.  **No Retry Mechanism**: The system does not implement retry logic for failed audit writes.
4.  **No Persistence Guarantee**: While the system attempts to log every operation, it does not guarantee 100% audit delivery in the event of audit storage failure.

## Constraints and Limitations

1. **Storage Growth:** Audit log grows monotonically.
2. **No Filtering:** All operations are logged.
3. **No Streaming:** Query-based, not real-time streaming.
4. **Source of Truth:** Audit log is sole source of history for removed reactions.

## Acceptance Criteria

1. **AC1:** Every ADD operation creates an audit entry
2. **AC2:** Every REPLACE operation creates an audit entry
3. **AC3:** Every REMOVE operation creates an audit entry
4. **AC4:** Audit entries include all required fields
5. **AC5:** Timestamps recorded in UTC with millisecond precision
6. **AC6:** Audit entries are immutable
7. **AC7:** Audit entries can be queried by user_id
8. **AC8:** Audit entries can be queried by entity
9. **AC9:** Audit queries support date range filtering
10. **AC10:** Audit storage can be configured independently
11. **AC11:** Audit storage failures do not impact reaction operations
12. **AC12:** NullAuditor is available as no-op implementation
