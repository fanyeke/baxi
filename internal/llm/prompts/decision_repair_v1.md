# Decision Repair Prompt

The previous decision output had validation errors. Please fix them and produce a valid output.

## Original Context
The decision context/trigger/object information is unchanged from the previous request. Use the same context as before.

## Validation Errors
{{range .Errors}}
- {{.Field}}: {{.Message}}
{{end}}

## Instructions
1. Only fix the fields listed above
2. Keep all other fields unchanged
3. Output MUST be valid "decision_output.v1" JSON only
4. All recommended_actions must be in allowed_actions
5. requires_human_review must be true
