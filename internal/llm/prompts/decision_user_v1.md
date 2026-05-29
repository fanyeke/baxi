# Decision User Prompt Template v1

CONTEXT:
{{.ContextJSON}}

{{if .EnrichedObjects}}
### Related Objects (via Ontology Links)
{{range .EnrichedObjects}}
Related {{.ObjectType}} ({{.ObjectID}} via "{{.LinkName}}"):
{{range $k, $v := .Properties}}  {{$k}}: {{$v}}
{{end}}
{{end}}
{{end}}

ALLOWED ACTIONS:
{{range .AllowedActions}}- {{.}}
{{end}}

FORBIDDEN ACTIONS:
{{range .ForbiddenActions}}- {{.}}
{{end}}

Generate a structured JSON decision following the system instructions.
