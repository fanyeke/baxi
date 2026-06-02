package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
)


// ──── actionConfigToDefinition ─────────────────────────────────────────────

func TestActionConfigToDefinition_Basic(t *testing.T) {
	cfg := ActionConfig{
		Description: "Export a report",
		RiskLevel:   "low",
		AllowedBy:   []string{"admin", "analyst"},
		Adapter:     "feishu",
	}

	def := actionConfigToDefinition("export_report", cfg)
	assert.Equal(t, "export_report", def.Name)
	assert.Equal(t, "Export a report", def.Description)
	assert.Equal(t, "low", def.RiskLevel)
	assert.Equal(t, []string{"admin", "analyst"}, def.AllowedBy)
	assert.Equal(t, "feishu", def.Adapter)
	assert.NotNil(t, def.PayloadSchema)
	assert.Empty(t, def.PayloadSchema)
}

func TestActionConfigToDefinition_WithPayloadSchema(t *testing.T) {
	cfg := ActionConfig{
		Description: "Create followup task",
		RiskLevel:   "medium",
		PayloadSchemaRaw: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{"type": "string"},
			},
		},
	}

	def := actionConfigToDefinition("create_followup_task", cfg)
	assert.NotNil(t, def.PayloadSchema)
	assert.Equal(t, "medium", def.RiskLevel)
	props, ok := def.PayloadSchema["properties"]
	assert.True(t, ok)
	assert.NotNil(t, props)
}

func TestActionConfigToDefinition_NilPayloadSchemaDefaultsToEmpty(t *testing.T) {
	cfg := ActionConfig{
		Description:      "Notify owner",
		PayloadSchemaRaw: nil,
	}

	def := actionConfigToDefinition("notify_owner", cfg)
	assert.NotNil(t, def.PayloadSchema)
	assert.Empty(t, def.PayloadSchema)
}

// ──── NewActionSchemaCatalog ───────────────────────────────────────────────

func TestNewActionSchemaCatalog_NonNil(t *testing.T) {
	reg := NewEmptyRegistry()
	catalog := NewActionSchemaCatalog(reg)
	assert.NotNil(t, catalog)
}

// ──── ListActionSchemas ────────────────────────────────────────────────────

func TestListActionSchemas_EmptyRegistry(t *testing.T) {
	reg := NewEmptyRegistry()
	catalog := NewActionSchemaCatalog(reg)

	defs, err := catalog.ListActionSchemas()
	assert.NoError(t, err)
	assert.Empty(t, defs)
}

func TestListActionSchemas_WithActions(t *testing.T) {
	reg := &ActionRegistry{
		whitelist: map[string]bool{"export_report": true, "notify_owner": true},
		config: &ActionRegistryConfig{
			Actions: map[string]ActionConfig{
				"export_report": {
					Description: "Export report",
					RiskLevel:   "low",
					Adapter:     "feishu",
				},
				"notify_owner": {
					Description: "Notify owner",
					RiskLevel:   "medium",
					Adapter:     "feishu",
				},
			},
		},
	}

	catalog := NewActionSchemaCatalog(reg)
	defs, err := catalog.ListActionSchemas()

	assert.NoError(t, err)
	assert.Len(t, defs, 2)

	names := make([]string, 0, len(defs))
	for _, d := range defs {
		names = append(names, d.Name)
	}
	assert.Contains(t, names, "export_report")
	assert.Contains(t, names, "notify_owner")
}

// ──── GetActionSchema ──────────────────────────────────────────────────────

func TestGetActionSchema_Found(t *testing.T) {
	reg := &ActionRegistry{
		whitelist: map[string]bool{"export_report": true},
		config: &ActionRegistryConfig{
			Actions: map[string]ActionConfig{
				"export_report": {
					Description: "Export report",
					RiskLevel:   "low",
					Adapter:     "feishu",
				},
			},
		},
	}

	catalog := NewActionSchemaCatalog(reg)
	def, err := catalog.GetActionSchema("export_report")

	assert.NoError(t, err)
	assert.NotNil(t, def)
	assert.Equal(t, "export_report", def.Name)
}

func TestGetActionSchema_NotFound(t *testing.T) {
	reg := NewEmptyRegistry()
	catalog := NewActionSchemaCatalog(reg)

	def, err := catalog.GetActionSchema("nonexistent_action")
	assert.NoError(t, err)
	assert.Nil(t, def)
}
