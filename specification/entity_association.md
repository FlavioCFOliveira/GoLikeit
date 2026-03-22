# Entity Association Specification

## Overview

The system provides an entity-agnostic mechanism for associating reactions with any type of content using a **Reaction Target** - a composite identifier of entity type and entity instance.

## Core Concepts

### Reaction Target

A **Reaction Target** is the unique combination of `(entity_type, entity_id)`.

**Properties:**
- **Entity Type:** Categorizes content type (e.g., "photo", "article")
- **Entity ID:** Uniquely identifies an instance (e.g., "123")
- **Uniqueness:** The combination forms a unique key
- **Isolation:** Reactions on different targets are independent

**Examples:**
- `("photo", "123")` - A specific photo
- `("article", "abc-456")` - A specific article

## Functional Requirements

### Requirement 1: Entity Type Identification

The system supports arbitrary entity types.

**Constraints:**
- Non-empty strings
- Maximum 64 characters
- Pattern: `[a-zA-Z0-9_-]+`
- No registry maintained; validation is syntactic

### Requirement 2: Entity Instance Identification

The system supports unique instance identifiers.

**Constraints:**
- Non-empty strings
- Maximum 256 characters
- Treated as opaque values

### Requirement 3: Reaction Target Uniqueness

The composite key uniquely identifies what is being reacted to.

- Reactions on different targets are isolated
- Each target maintains independent counts per reaction type

### Requirement 4: Reaction Target Agnostic Operations

All operations work uniformly regardless of target composition.

- No entity-type-specific configuration required
- Adding new entity types requires no module changes

### Requirement 5: Reaction Target Existence Independence

The system does not validate target existence.

- Reactions may be recorded for non-existent targets
- Existence validation is the responsibility of the consuming application

## Constraints and Limitations

1. **Identifier Length:** Entity type max 64 chars; entity ID max 256 chars.
2. **No Hierarchical Relationships:** No parent-child relationships between targets.
3. **No Metadata:** Only entity type and ID are stored.
4. **String Identifiers Only:** Components must be representable as strings.
5. **Target Immutability:** Target components cannot be changed after recording.

## Acceptance Criteria

1. **AC1:** Accepts valid entity types (alphanumeric, underscores, hyphens, max 64)
2. **AC2:** Rejects empty entity types
3. **AC3:** Rejects entity types exceeding 64 characters
4. **AC4:** Accepts non-empty entity IDs (max 256)
5. **AC5:** Rejects entity IDs exceeding 256 characters
6. **AC6:** Reactions on different targets are independent
7. **AC7:** Entity types are case-sensitive
8. **AC8:** Entity IDs treated as opaque
9. **AC9:** No configuration required to add new entity types
