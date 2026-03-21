# Entity Association Specification

## Overview

The system shall provide an entity-agnostic mechanism for associating reactions with any type of content in the application. The module treats all target content uniformly using a **Reaction Target** - a composite identifier consisting of an entity type and an entity instance identifier, enabling reactions on photos, articles, comments, or any other content types without modification.

## Core Concepts

### Reaction Target

A **Reaction Target** is the unique combination of `(entity_type, entity_id)` that identifies what is being reacted to.

**Properties:**
- **Entity Type Component:** Categorizes the type of content (e.g., "photo", "article", "comment")
- **Entity ID Component:** Uniquely identifies a specific instance within the type (e.g., "123", "abc-456")
- **Uniqueness:** The combination `(entity_type, entity_id)` forms a unique key for reaction targeting
- **Isolation:** Reactions on different Reaction Targets are completely independent

**Examples:**
- `("photo", "123")` - A specific photo with ID 123
- `("article", "abc-456")` - A specific article with ID abc-456
- `("comment", "789")` - A specific comment with ID 789

**String Representation:**
- Reaction Targets may be represented as `"entity_type:entity_id"` for logging and display purposes
- Examples: `"photo:123"`, `"article:abc-456"`

## Functional Requirements

### Requirement 1: Entity Type Identification

**Description:** The system shall support arbitrary entity types defined by the consuming application as part of a Reaction Target.

**Inputs:**
- Entity type identifier (string, case-sensitive)

**Outputs:**
- Validation result for the entity type format

**Constraints:**
- Entity type identifiers shall be non-empty strings
- Entity type identifiers shall have a maximum length of 64 characters
- Entity type identifiers shall consist of alphanumeric characters, underscores, and hyphens
- The system shall not maintain a registry of valid entity types; validation is purely syntactic

**Behavior:**
- The system shall accept any syntactically valid entity type identifier
- The system shall reject empty or malformed entity type identifiers
- Different entity types are treated as completely separate namespaces within Reaction Targets

### Requirement 2: Entity Instance Identification

**Description:** The system shall support unique instance identifiers within each entity type as part of a Reaction Target.

**Inputs:**
- Entity instance identifier (string)
- Entity type context (from the Reaction Target)

**Outputs:**
- Validation result for the entity identifier format

**Constraints:**
- Entity instance identifiers shall be non-empty strings
- Entity instance identifiers shall have a maximum length of 256 characters
- The system shall treat entity identifiers as opaque values (content-agnostic)

**Behavior:**
- The system shall accept any non-empty entity identifier within length limits
- Identifiers are unique within their entity type (same identifier in different types refers to different Reaction Targets)
- The system shall not interpret or validate the semantic meaning of identifiers

### Requirement 3: Reaction Target Uniqueness

**Description:** The system shall use a Reaction Target - the composite key of (entity_type, entity_id) - to uniquely identify what is being reacted to.

**Requirements:**
- The combination of entity_type and entity_id shall uniquely identify a Reaction Target
- Reactions on different Reaction Targets shall be completely isolated
- Each Reaction Target maintains its own independent reaction counts and state

**Behavior:**
- LIKE on Reaction Target `("photo", "123")` is independent of LIKE on `("article", "123")`
- LIKE on Reaction Target `("photo", "123")` is independent of LIKE on `("photo", "456")`
- Query operations shall require both entity_type and entity_id to identify a Reaction Target
- Reaction Targets are the atomic unit of reaction aggregation

### Requirement 4: Reaction Target Agnostic Operations

**Description:** All reaction operations shall work uniformly regardless of Reaction Target composition.

**Requirements:**
- LIKE, UNLIKE, DISLIKE, and UNDISLIKE operations shall behave identically for all Reaction Targets
- Count aggregation shall be available per Reaction Target
- Cross-Reaction-Target aggregation is not required

**Behavior:**
- The system shall not require entity-type-specific configuration
- Adding support for a new entity type (and thus new Reaction Targets) requires no changes to the reaction module
- Statistics and queries shall support filtering by entity type component of Reaction Targets

### Requirement 5: Reaction Target Existence Independence

**Description:** The system shall not require Reaction Targets to correspond to existing entities in order to record reactions.

**Requirements:**
- Reactions may be recorded for Reaction Targets that do not exist in the application
- The system shall not validate entity existence against external systems
- Reactions on non-existent Reaction Targets shall behave identically to reactions on existent ones

**Rationale:**
- The reaction module maintains separation of concerns
- Reaction Target existence validation is the responsibility of the consuming application
- This design enables pre-creation reactions or post-deletion cleanup strategies

## Constraints and Limitations

1. **Identifier Length:** Entity type identifiers are limited to 64 characters; entity instance identifiers are limited to 256 characters. These constraints apply to the components of a Reaction Target.

2. **No Hierarchical Relationships:** The system does not recognize or enforce parent-child relationships between Reaction Targets. Reactions on one Reaction Target do not affect others.

3. **No Reaction Target Metadata:** The system stores only the entity type and identifier components of a Reaction Target, not any descriptive metadata about the underlying entity (titles, descriptions, etc.).

4. **Namespace Isolation:** Reactions are strictly namespaced by the entity type component of Reaction Targets. No cross-type queries or aggregations are supported.

5. **String Identifiers Only:** Reaction Target components must be representable as strings. Binary or complex identifier types must be serialized by the consuming application.

6. **Reaction Target Immutability:** Once a reaction is recorded for a Reaction Target, the target components cannot be changed. Migration of reactions between targets must be handled by the consuming application.

## Relationships with Other Functional Blocks

- **[reaction_management.md](reaction_management.md):** Defines the reaction operations that target entities
- **[data_persistence.md](data_persistence.md):** Defines how entity references are stored
- **[api_interface.md](api_interface.md):** Defines how entity references are passed through the public interface

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version of entity association specification |
| 2026-03-21 | Update | Introduced Reaction Target concept; updated all requirements to use Reaction Target terminology |

## Acceptance Criteria

1. **AC1:** The system accepts any syntactically valid entity type component (alphanumeric, underscores, hyphens, max 64 chars)
2. **AC2:** The system rejects empty entity type components
3. **AC3:** The system rejects entity type components exceeding 64 characters
4. **AC4:** The system accepts any non-empty entity instance identifier (max 256 chars)
5. **AC5:** The system rejects entity instance identifiers exceeding 256 characters
6. **AC6:** Reactions on Reaction Target `("type_a", "id_123")` are independent of reactions on `("type_b", "id_123")`
7. **AC7:** Reactions on Reaction Target `("type_a", "id_123")` are independent of reactions on `("type_a", "id_456")`
8. **AC8:** Entity type components are case-sensitive ("Photo" and "photo" are different types, thus different Reaction Targets)
9. **AC9:** Entity identifier components are treated as opaque values (no interpretation of content)
10. **AC10:** No configuration is required to add support for new Reaction Targets with new entity types
