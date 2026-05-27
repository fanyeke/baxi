package llm

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed prompts/decision_repair_v1.md
var repairPromptContent string

// RepairPromptRenderer renders repair prompts from validation errors.
type RepairPromptRenderer struct {
	tmpl *template.Template
}

// RepairPromptData is the data passed to the repair prompt template.
type RepairPromptData struct {
	Errors []ValidationError
}

// NewRepairPromptRenderer creates a RepairPromptRenderer from the embedded repair prompt.
func NewRepairPromptRenderer() (*RepairPromptRenderer, error) {
	tmpl, err := template.New("repair").Parse(repairPromptContent)
	if err != nil {
		return nil, fmt.Errorf("parse repair prompt: %w", err)
	}
	return &RepairPromptRenderer{tmpl: tmpl}, nil
}

// RenderRepairPrompt renders the repair prompt with the given validation errors.
func (r *RepairPromptRenderer) RenderRepairPrompt(errors []ValidationError) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, RepairPromptData{Errors: errors}); err != nil {
		return "", fmt.Errorf("render repair prompt: %w", err)
	}
	return buf.String(), nil
}
