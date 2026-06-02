# Feature Specification: Ontology v2 Productionization & E2E

**Feature Branch**: `002-ontology-v2-productionization`  
**Created**: 2026-06-02  
**Status**: Complete  
**Input**: User description: "Ontology v2 Productionization & E2E - Fix MCP wiring, action safety, and end-to-end validation"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recipe-Driven Context Building (Priority: P1)

As a Pi Agent operator, I want to build context for a decision case using recipe-driven configuration, so that the agent receives a complete, evidence-backed context envelope with governance rules and allowed actions.

**Why this priority**: This is the foundational capability that enables the Agent to make informed decisions. Without it, the Agent lacks the structured context required for AIP-style operations.

**Independent Test**: Can be fully tested by calling `build_context(case_id, recipe_id?)` and verifying the returned `LLMSafeContextEnvelope` contains `context_hash`, `evidence`, `object_context`, `allowed_actions`, `governance`, and `redaction_summary`.

**Acceptance Scenarios**:

1. **Given** a running baxi-mcp server with v2 ontology objects loaded, **When** the Agent calls `build_context` with a valid case_id, **Then** the system returns a complete LLMSafeContextEnvelope with all required fields populated.
2. **Given** a case where no specific recipe is configured, **When** the Agent calls `build_context` with only case_id, **Then** the system falls back to the object's default recipe and still returns a valid context envelope.
3. **Given** a case where build_context service is not available, **When** the Agent calls `build_context`, **Then** the system returns a clear error indicating the service is unavailable rather than failing silently.

---

### User Story 2 - One-to-Many Object Relationship Queries (Priority: P1)

As a Pi Agent operator, I want to query linked objects using the v2 LinkResolver (e.g., seller → recent_orders), so that I can retrieve multiple related records instead of being limited to single-value relationships.

**Why this priority**: The v2 ontology supports rich one-to-many relationships. Without proper LinkResolver wiring, the Agent cannot access critical related data like a seller's order history, severely limiting decision quality.

**Independent Test**: Can be fully tested by calling `get_linked_objects(seller, seller_id, recent_orders)` and verifying it returns an array of order records rather than a single value or an error.

**Acceptance Scenarios**:

1. **Given** a seller object with linked recent_orders configured in v2 schema, **When** the Agent calls `get_linked_objects` for recent_orders, **Then** the system returns an array of order objects.
2. **Given** an object type where v2 LinkResolver is not configured, **When** the Agent calls `get_linked_objects`, **Then** the system gracefully falls back to the legacy v1 Via model.
3. **Given** a request for a non-existent link, **When** the Agent calls `get_linked_objects`, **Then** the system returns an appropriate empty result or error message.

---

### User Story 3 - Safe Action Execution with Approval Workflow (Priority: P1)

As a platform operator, I want action execution to follow a proper proposal → approval → execute workflow, so that no action can bypass review or execute uncontrolled in production.

**Why this priority**: This is a critical security and governance requirement. The current behavior auto-approves and executes actions with dry_run=false, which violates AIP principles and creates operational risk.

**Independent Test**: Can be fully tested by attempting to execute an action through the MCP and verifying that: (a) `propose_action` creates a proposal in pending_review state, (b) `execute_action` defaults to dry-run, and (c) unapproved actions cannot be executed.

**Acceptance Scenarios**:

1. **Given** an action binding configured for an object, **When** the Agent calls `propose_action`, **Then** the system creates an action_proposal with status `proposed` or `pending_review`, not `approved`.
2. **Given** an action where `requires_approval=true`, **When** the Agent attempts direct execution without approval, **Then** the system rejects the request with a clear authorization error.
3. **Given** an approved action proposal, **When** the Agent calls `execute_proposal`, **Then** the system executes the action in dry-run mode by default, and only executes for real when explicitly configured.
4. **Given** an unapproved action proposal, **When** the Agent calls `execute_proposal`, **Then** the system rejects the execution attempt.

---

### User Story 4 - End-to-End MCP Validation (Priority: P2)

As a platform engineer, I want to run a complete MCP end-to-end test for the seller_late_delivery_alert scenario, so that I can verify the entire Ontology v2 pipeline works correctly from query to action proposal.

**Why this priority**: E2E validation ensures all components work together. This is essential before expanding to more objects or releasing to production.

**Independent Test**: Can be fully tested by running the documented E2E test sequence and verifying each step produces the expected output.

**Acceptance Scenarios**:

1. **Given** a fresh baxi-mcp instance with seller_late_delivery_alert fixtures, **When** running the E2E test sequence (`describe_ontology` → `get_object` → `get_linked_objects` → `build_context` → `propose_action` → `approve_proposal` → `execute_proposal(dry_run=true)`), **Then** all steps complete successfully with valid outputs.
2. **Given** the E2E test sequence, **When** verifying the quickstart documentation, **Then** a new user can follow the quickstart and reproduce the same results.

---

### Edge Cases

- What happens when `build_context` is called with a non-existent case_id?
- What happens when `get_linked_objects` is called for an object type that exists only in v1?
- What happens when `propose_action` is called for an action that is not bound to the object?
- What happens when `execute_action` is called with `requires_approval=true` but no approval exists?
- What happens when the metric_definitions.yml or context_recipes.yml files are missing or malformed?
- How does the system handle v1/v2 object coexistence conflicts?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST wire the `buildContextSvc` into the MCP server startup path so that `build_context` is fully operational when v2 ontology objects are present.
- **FR-002**: The system MUST implement and register a `propose_action` MCP tool that creates action proposals in a pending review state without executing them.
- **FR-003**: The system MUST modify `execute_action` to default to dry-run mode and MUST NOT automatically create approved proposals.
- **FR-004**: The system MUST enforce that actions with `requires_approval=true` cannot be executed without proper approval.
- **FR-005**: The system MUST wire the v2 `LinkResolver` into `get_linked_objects` so that one-to-many relationships (e.g., seller → recent_orders) return arrays of linked objects.
- **FR-006**: The system MUST maintain backward compatibility by falling back to v1 Via-model when v2 LinkResolver is unavailable for an object type.
- **FR-007**: The system MUST load `metric_definitions.yml` and `context_recipes.yml` at startup and use them for recipe-driven context building.
- **FR-008**: The system MUST ensure `execute_proposal` only processes proposals that have been explicitly approved.
- **FR-009**: The system MUST provide an E2E test script that validates the complete seller_late_delivery_alert MCP workflow.
- **FR-010**: The system MUST update documentation (quickstart, AGENTS.md, tool contracts) to reflect the productionized v2 behavior.

### Key Entities *(include if feature involves data)*

- **Ontology Object (v2)**: A semantic entity defined by schema v2 with properties, metrics, relationships, context recipes, and action bindings.
- **Context Recipe**: A configuration that defines how to build an LLM-safe context envelope for a given object type, including which metrics and evidence to include.
- **Action Proposal**: A request to execute an action on an object, created in `proposed`/`pending_review` state and requiring approval before execution.
- **LinkResolver**: A service that resolves object relationships using v2 schema definitions, supporting one-to-many cardinality.
- **LLMSafeContextEnvelope**: The structured output of `build_context` containing context_hash, evidence, object_context, allowed_actions, governance, and redaction_summary.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The baxi-mcp server starts with a non-nil `buildContextSvc` when v2 objects are configured.
- **SC-002**: A call to `build_context(case_id)` returns a complete `LLMSafeContextEnvelope` with all 6 required fields populated within 2 seconds.
- **SC-003**: A call to `get_linked_objects(seller, seller_id, recent_orders)` returns an array of at least one order record when the seller has orders.
- **SC-004**: The `propose_action` tool exists in the MCP toolset and creates proposals in `proposed`/`pending_review` state 100% of the time.
- **SC-005**: Direct `execute_action` calls default to dry-run mode and do not modify production state without explicit configuration.
- **SC-006**: Unapproved proposals are rejected by `execute_proposal` with a clear authorization error 100% of the time.
- **SC-007**: The E2E test script covers the full workflow: describe → get_object → get_linked_objects → build_context → propose_action → approve → execute_proposal(dry_run=true), and completes without errors.
- **SC-008**: A new user can follow the quickstart documentation and reproduce the seller_late_delivery_alert end-to-end workflow in under 15 minutes.

## Assumptions

- The Ontology v2 core code (schema, parser, compiler, registry) is already implemented and committed.
- The v1 ontology system remains operational during this phase; v1/v2 coexistence is maintained.
- PostgreSQL is running and accessible for MCP server startup.
- The `seller_late_delivery_alert` scenario is the primary validation target; other objects will be migrated in a subsequent phase.
- Pi Agent integration uses stdio MCP transport.
- Action bindings for `seller` object (e.g., `notify_owner`) are already configured in YAML.
- The existing test infrastructure (testcontainers for PostgreSQL) is available for E2E tests.
