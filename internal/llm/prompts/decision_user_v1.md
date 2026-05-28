# Decision User Prompt Template v1

CONTEXT:
{{.ContextJSON}}

ALLOWED ACTIONS:
{{range .AllowedActions}}- {{.}}
{{end}}

FORBIDDEN ACTIONS:
{{range .ForbiddenActions}}- {{.}}
{{end}}

Generate a structured JSON decision following the system instructions.
