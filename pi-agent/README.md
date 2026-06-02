# Pi Agent

Read-only AI decision-support agent for e-commerce operations. The Pi Agent connects to the Baxi MCP server, reads ontology and decision context, inspects evidence, and generates structured JSON decisions for platform operators.

## Architecture

```
Pi Agent (LLM)  ──stdio MCP──>  baxi-mcp  ──>  PostgreSQL (ontology v2, alerts, cases)
     │
     └── outputs: structured JSON decision (case_id, severity, confidence, recommended_actions)
```

The agent communicates with `baxi-mcp` over the stdio MCP transport. All data is read-only -- the agent inspects ontology objects, decision context envelopes, and linked evidence to produce recommendations. It never writes to the database, executes actions, or approves proposals.

## Directory Layout

```
pi-agent/
├── prompts/
│   ├── system.md          # System prompt: role, allowed tools, workflow, output schema
│   └── tool-policy.md     # Tool allow/deny list, conditional rules, rate limits, context budget
├── schemas/
│   ├── decision.schema.json   # Canonical JSON schema for decision output validation
│   └── decision.schema_test.go
├── golden_cases/
│   ├── case-01-seller-late-delivery.json
│   ├── case-02-product-review-drop.json
│   ├── case-03-category-gmv-drop.json
│   ├── case-04-region-delivery-anomaly.json
│   └── case-05-customer-value-anomaly.json
├── eval/
│   ├── evaluator.go          # Go evaluator: scores decisions 0-100 against golden cases
│   └── evaluator_test.go
├── docs/
│   └── mcp-calls.md          # MCP tool call reference with examples and error guidance
└── README.md                 # This file
```

### prompts/

- **system.md** -- Full system prompt loaded into the agent at startup. Defines the agent's role (read-only decision support), the six allowed tools (`describe_ontology`, `get_object`, `get_linked_objects`, `build_context`, `get_decision_context`, `list_action_schemas`), the five-step decision workflow, safety rules, and the JSON output schema.
- **tool-policy.md** -- Detailed tool governance: explicit allow/deny lists for all 30+ MCP tools, conditional rules (e.g., `max_depth <= 2` for `get_linked_objects`), per-phase rate limits (max 11 calls per decision), and a 12k-token context budget.

### schemas/

- **decision.schema.json** -- JSON Schema (draft-07) that every agent decision output must validate against. Required fields: `schema_version`, `decision_type` (one of `monitor_only`, `investigate`, `optimize`, `intervention`, `experiment`), `severity`, `summary`, `rationale`, `recommended_actions`, `confidence`, and `requires_human_review` (must be `true`).

### golden_cases/

Five reference decision cases used to evaluate agent correctness:

| Case | Scenario |
|------|----------|
| case-01 | Seller late delivery rate 31% (baseline 8%) |
| case-02 | Product review score drop from 4.2 to 2.8 |
| case-03 | Category GMV drop 18% week-over-week |
| case-04 | Regional delivery anomaly with 45% late rate |
| case-05 | Customer lifetime value anomaly (CLV drop 40%) |

Each golden case contains the context summary, expected decision output, and grading criteria (severity match, minimum confidence, must/must-not recommend actions, required evidence refs).

### eval/

Go evaluator that scores an agent-produced decision against a golden case. Scoring breakdown:

| Category | Weight | What It Measures |
|----------|--------|------------------|
| Schema | 25% | Valid schema_version, decision_type, severity, confidence, non-empty summary/rationale/actions, requires_human_review=true |
| Evidence | 25% | Required evidence_refs present, correctly formatted (`metric:` or `link:` prefix) |
| Actions | 25% | Must-recommend actions present, must-not-recommend/forbidden actions absent |
| Severity/Confidence | 25% | Severity matches expected, confidence meets minimum threshold, high-risk + low-confidence penalty |

**Pass threshold: >= 80/100.** The evaluator also applies penalties for contradictory pairings (high/critical severity with confidence < 0.7).

### docs/

- **mcp-calls.md** -- Complete reference for all six allowed MCP tools, each with parameter definitions, example JSON-RPC requests and responses, usage guidance in the decision workflow, and common mistakes.

## Quick Start

### 1. Start baxi-mcp

```bash
/tmp/baxi-mcp
```

This starts the MCP server on stdio, ready to accept agent connections.

### 2. Configure the Pi Agent

Set the agent's system prompt to `prompts/system.md` and apply the tool policy from `prompts/tool-policy.md`. These files define the agent's identity, constraints, and workflow.

### 3. Run a Decision

The agent follows a five-step workflow:

```
Step 1: describe_ontology        -- discover object types, metrics, links
Step 2: build_context(case_id)   -- fetch the evidence envelope
Step 3: get_linked_objects       -- drill into related records
Step 4: analyze evidence         -- compare metrics, assess severity
Step 5: output decision JSON     -- single valid JSON object
```

### 4. Evaluate

```bash
cd eval
go test -v ./...
```

This runs the evaluator tests, including `TestLoadGoldenCase` which validates all golden case files load correctly and `TestEvaluate_ValidDecision` which verifies a correct decision scores >= 99/100.

## Decision Flow (Detail)

1. **describe_ontology** -- Call once at session start. Returns all registered AIP object types (seller, order, product, category, region, customer, etc.) with their properties, computed metrics, relationship links, and allowed action bindings.

2. **build_context(case_id)** -- Primary context-gathering step. Returns an `LLMSafeContextEnvelope` containing the trigger alert, root object properties, evidence array (metrics + link references), allowed/forbidden actions, governance metadata, redaction summary, and versioned config hashes.

3. **get_linked_objects** (conditional) -- Drill into evidence links returned by `build_context`. Only when aggregate metrics are insufficient. Limited to 3 calls per decision, `max_depth <= 2`.

4. **get_decision_context(case_id)** (conditional) -- Supplementary context when `build_context` governance details are insufficient.

5. **list_action_schemas** -- Understand payload requirements and risk levels of available actions before making recommendations. Call once per session.

6. **Output decision JSON** -- Must validate against `schemas/decision.schema.json`. Fields: `case_id`, `context_hash`, `decision` (approve/monitor/escalate/insufficient_evidence/no_action), `confidence` (0.0-1.0), `severity` (low/medium/high/critical), `reasoning`, `evidence_refs`, `recommended_actions`, `requires_human_approval`.

## Safety

The Pi Agent enforces these invariants:

- **Read-only tools only.** Of the 30+ tools exposed by baxi-mcp, only 6 are allowed. All write-capable and execution tools are explicitly denied.
- **requires_human_approval must always be true** for any state-changing action recommendation. The agent recommends; a human approves.
- **No direct execution.** The agent never calls `execute_action`, `propose_action`, `approve_proposal`, or any execution-path tool. Even `dry_run` is forbidden.
- **Evidence grounding.** Every claim in the output JSON must reference a specific evidence key from `build_context`. Fabricated references are a violation.
- **Low-confidence guard.** If confidence < 0.5, the agent may only recommend `monitor` with no actions.
- **Governance-aware.** Redacted fields (PII, L2/L3 classifications) must not be speculated about.

## Evaluation

```bash
cd eval && go test -v -run TestEvaluate
```

The evaluator loads all five golden cases, runs a decision against each, and produces a breakdown:

```
Score: 92.5/100 (PASS)
  Schema:    22.0/25
  Evidence:  24.0/25
  Actions:   23.0/25
  Severity:  23.5/25
```

**Pass threshold: >= 80/100.** A failing evaluation indicates one or more of: schema non-compliance, missing evidence references, forbidden action recommendations, or severity/confidence mismatches.
