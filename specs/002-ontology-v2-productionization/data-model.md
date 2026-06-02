# Data Model: Ontology v2 Productionization & E2E

**Feature**: specs/002-ontology-v2-productionization
**Date**: 2026-06-02

## Entities

### ContextRecipe

A declarative configuration that defines how to construct an LLM-safe context envelope for a specific object type and scenario.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Unique recipe identifier (e.g., `seller_late_delivery_alert`) |
| ObjectType | string | Target ontology object type |
| Trigger | RecipeTrigger | Conditions that activate this recipe |
| Include | RecipeInclude | Data sources to include (metrics, links, actions, context) |
| Budget | RecipeBudget | Limits on context size (max_links, max_metrics, max_chars) |
| Governance | RecipeGovernance | Rules for redaction, required_fields, banned_fields |

**State**: Immutable after load. Loaded from `config/context_recipes.yml` at startup.

**Validation Rules**:
- Name must be unique across all recipes
- ObjectType must reference a registered v2 object type
- Budget limits must be positive integers
- Governance banned_fields must not overlap with required_fields

---

### ActionProposal

A request to execute an action on an ontology object, subject to review and approval.

| Field | Type | Description |
|-------|------|-------------|
| ProposalID | string | UUIDv4, primary key |
| CaseID | string | References the decision case that triggered this proposal |
| DecisionID | *string | References the specific decision within the case |
| ActionType | string | Canonical action type (e.g., `notify_owner`) |
| Payload | jsonb | Action parameters as JSON |
| ApplyStatus | string | Enum: `proposed`, `approved`, `rejected`, `applying`, `applied`, `failed` |
| CreatedAt | timestamp | Proposal creation time |
| Title | string | Human-readable title |
| Description | *string | Detailed description |
| RiskLevel | *string | `low`, `medium`, `high` |
| RequiresHumanReview | bool | Whether manual approval is required |

**State Transitions**:
```
proposed ──[approve]──→ approved ──[execute]──→ applying ──[success]──→ applied
   │                       │                                    │
   │                       │                                    └──[failure]──→ failed
   │                       │
   └──[reject]──→ rejected  └──[cancel]──→ rejected
```

**Validation Rules**:
- CaseID must reference an existing decision case
- ActionType must be registered in the global ActionRegistry
- Payload must validate against the action's JSON schema
- ApplyStatus transitions must follow the state machine above

---

### ObjectTypeV2

A semantic entity definition in the v2 ontology schema.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique object type identifier (e.g., `seller`) |
| DisplayName | string | Human-readable name |
| PrimaryKey | string | Database primary key field |
| Properties | []PropertyV2 | Object attributes |
| Metrics | []MetricReference | Computed metrics |
| Links | []ObjectLinkV2 | Relationships to other objects |
| ContextRecipes | []string | Referenced recipe names |
| ActionBindings | []ActionBinding | Allowed actions for this object |

**Relationships**:
- Links → ObjectTypeV2 (many-to-many via LinkResolver)
- ContextRecipes → ContextRecipe (one-to-many via recipe name)
- ActionBindings → ActionRegistry entry (one-to-one via action type)

---

### ObjectLinkV2

A relationship definition between two v2 ontology objects.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Link identifier (e.g., `recent_orders`) |
| TargetType | string | Target object type ID |
| Cardinality | string | `one_to_one`, `one_to_many`, `many_to_many` |
| Strategy | string | `direct_key`, `reverse_lookup`, `bridge_table`, `query_ref` |
| SourceKey | string | Source field for join |
| Target | LinkTarget | Target schema/table/key configuration |
| Limit | int | Max results for one-to-many |
| Sort | []SortField | Result ordering |
| Fields | []string | Fields to select |

---

### LLMSafeContextEnvelope

The structured output returned by `build_context`.

| Field | Type | Description |
|-------|------|-------------|
| ContextHash | string | SHA-256 hash of canonicalized context |
| Evidence | []EvidenceItem | Supporting data points (metrics, links) |
| ObjectContext | map[string]interface{} | The resolved object with properties |
| AllowedActions | []ActionContract | Actions available for this object |
| Governance | GovernanceSummary | Applied redaction and field rules |
| RedactionSummary | []string | List of redacted fields and reasons |

---

## Key Relationships

```
ObjectTypeV2 --(Links)--> ObjectLinkV2 --(resolves via)--> LinkResolver
    │                                              │
    │(ContextRecipes)                              │(one_to_many)
    ↓                                              ↓
ContextRecipe --(drives)--> RecipeContextBuilder --(builds)--> LLMSafeContextEnvelope
    │                                                            │
    │(ActionBindings)                                            │(AllowedActions)
    ↓                                                            ↓
ActionBinding --(validates via)--> ActionBindingValidator    ActionContract
    │
    ↓
ActionProposal --(creates)--> propose_action --(approves)--> ReviewService
```

## Data Flow: build_context

```
case_id ──→ RecipeContextBuilder.BuildEnvelope()
                ├── caseSvc.GetCaseByID() → DecisionCase
                ├── compiler.ResolveObjectQuery() → ObjectContext
                ├── metricQuery.Resolve() → Metrics
                ├── linkExec.ExecuteLinkQuery() → Linked Objects
                ├── actionTypes.GetAllowedActions() → ActionContracts
                └── recipes[recipe_id] → ContextRecipe (governance rules)
                                    ↓
                            LLMSafeContextEnvelope
```

## Data Flow: get_linked_objects (v2)

```
object_type, object_id, link_name ──→ handleGetLinkedObjects()
                ├── Try s.linkResolver.GetLinkedObjects()
                │       ├── Compile SQL plan (strategy: direct_key/reverse_lookup/...)
                │       ├── Execute via pool.Query()
                │       └── Return []LinkedObject (array for one_to_many)
                └── Fallback: s.ontologySvc.GetLinkedObjects() (v1 Via model)
                            ├── registry.GetLinks() → ObjectLink[]
                            ├── querySvc.BuildObjectContext() → source properties
                            └── Return single target object
```

## Data Flow: propose_action

```
object_type, object_id, action_type, params ──→ propose_action handler
                ├── ActionBindingValidator.Validate(object_type, action_type)
                ├── caseSvc.GetCaseByID() or create new case
                ├── Build ActionProposalRow{ApplyStatus: "proposed"}
                ├── repo.CreateProposal(ctx, pool, row)
                └── Return {proposal_id, status: "proposed"}
```

## Validation Rules Summary

| Entity | Rule | Enforcement |
|--------|------|-------------|
| ContextRecipe | Name uniqueness | Startup load validation |
| ContextRecipe | ObjectType exists in registry | Startup load validation |
| ActionProposal | ActionType in registry | `ActionBindingValidator` |
| ActionProposal | Payload matches schema | `ActionBindingValidator.ValidatePayload()` |
| ActionProposal | CaseID exists | Foreign key constraint + repository check |
| ObjectLinkV2 | TargetType exists in v2 objects | `LinkResolver` compilation error |
| ObjectLinkV2 | Cardinality matches strategy | `LinkResolver` compilation error |

## State Machines

### ActionProposal Lifecycle

```
                    ┌─────────────┐
                    │   proposed  │◄───── propose_action / GenerateProposals
                    └──────┬──────┘
              approve │    │ reject
                      ↓    ↓
               ┌────────┐ ┌──────────┐
               │approved│ │ rejected │
               └───┬────┘ └──────────┘
         execute  │
                   ↓
              ┌─────────┐
              │ applying│
              └────┬────┘
         success   │   failure
                   ↓
            ┌──────┴──────┐
            │ applied     │ failed
            └─────────────┘
```

### build_context Service Availability

```
                    ┌─────────────┐
                    │  startup    │
                    └──────┬──────┘
      recipes.yml exists  │  missing
                          ↓
               ┌────────────────────┐
               │ buildContextSvc    │ nil
               │ = RecipeContextBuilder│
               └────────────────────┘
                          │
         build_context call │
                          ↓
              ┌─────────────────────┐
              │ buildContextSvc     │ nil → "not available"
              │ BuildEnvelope()     │ non-nil → execute recipe
              └─────────────────────┘
```
