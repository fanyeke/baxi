# llm: LLM Provider Abstraction

**Branch:** main

## OVERVIEW
Provider abstraction layer for LLM-based decision making. Supports OpenAI-compatible APIs and rule-based fallback. 17 files.

Decision engine is exposed via MCP tools (`decide`, `create_decision_case`, etc.) — Pi Agent invokes LLM decisions through the MCP stdio transport. The LLM provider chain (OpenAI → validation → repair → rule fallback) runs inside the MCP handler's call path.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Add a provider | `provider.go` + `provider_factory.go` | Implement DecisionProvider interface |
| OpenAI integration | `openai_provider.go` | OpenAI-compatible API calls with structured JSON output |
| Rule-based fallback | `rule_provider.go` | Heuristic fallback when LLM unavailable or fails |
| Disabled mode | `disabled_provider.go` | Returns error when LLM_ENABLED=false |
| Prompt templates | `prompt_registry.go` | System/user prompt construction with template params |
| Schema validation | `schema_validator.go` | Validates DecisionOutput against expected schema fields |
| Repair prompt | `repair_prompt.go` | Renders repair prompt for validation retry |
| Audit logging | `audit.go` | Logs decision requests, completions, failures, fallbacks |
| Context envelope | `context_envelope.go` | LLMSafeContext + DecisionOutput type definitions |


## KEY PATTERNS

- **Provider abstraction**: DecisionProvider interface (single GenerateDecision method) allows swapping LLM backends
- **Structured output**: OpenAI provider requests `response_format: json_schema` for parseable decisions
- **Repair mechanism**: On validation failure, a repair prompt retries once before fallback
- **Disabled mode**: When LLM_ENABLED=false (default), factory returns DisabledProvider — only rule-based works
- **Validation**: schema_validator checks action types, required fields, enum values post-generation

## ANTI-PATTERNS

- disabled_provider.go duplicates logic from rule_provider.go — could consolidate into single no-op path
- Prompt templates hardcoded in Go code (no external .prompt file management)
- DecisionOutput.SchemaVersion only set to "decision_output.v1" in tests, not production
