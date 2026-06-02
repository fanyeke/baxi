package action

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMain sets up package-level test environment. Integration tests that call
// ExecuteProposal with dry_run=false require BAXI_ALLOW_LIVE_EXECUTION=true.
// Individual tests can override this with t.Setenv for negative testing.
func TestMain(m *testing.M) {
	orig := os.Getenv("BAXI_ALLOW_LIVE_EXECUTION")
	os.Setenv("BAXI_ALLOW_LIVE_EXECUTION", "true")
	code := m.Run()
	if orig == "" {
		os.Unsetenv("BAXI_ALLOW_LIVE_EXECUTION")
	} else {
		os.Setenv("BAXI_ALLOW_LIVE_EXECUTION", orig)
	}
	os.Exit(code)
}

// ──── actionChannel ────────────────────────────────────────────────────────

func TestActionChannel_KnownTypes(t *testing.T) {
	assert.Equal(t, "feishu", actionChannel("export_report"))
	assert.Equal(t, "feishu", actionChannel("notify_owner"))
	assert.Equal(t, "feishu", actionChannel("create_outbox_message"))
	assert.Equal(t, "github", actionChannel("create_followup_task"))
}

func TestActionChannel_UnknownType(t *testing.T) {
	assert.Equal(t, "unknown", actionChannel("unknown_action"))
}

func TestActionChannel_EmptyType(t *testing.T) {
	assert.Equal(t, "unknown", actionChannel(""))
}

// ──── mapRiskLevel ─────────────────────────────────────────────────────────

func TestMapRiskLevel_KnownSeverities(t *testing.T) {
	assert.Equal(t, "high", mapRiskLevel("critical"))
	assert.Equal(t, "high", mapRiskLevel("high"))
	assert.Equal(t, "medium", mapRiskLevel("medium"))
	assert.Equal(t, "low", mapRiskLevel("low"))
}

func TestMapRiskLevel_UnknownSeverity(t *testing.T) {
	assert.Equal(t, "medium", mapRiskLevel("unknown"))
	assert.Equal(t, "medium", mapRiskLevel(""))
}

// ──── isCanonical ──────────────────────────────────────────────────────────

func TestIsCanonical_CanonicalActions(t *testing.T) {
	assert.True(t, isCanonical("create_followup_task"))
	assert.True(t, isCanonical("export_report"))
	assert.True(t, isCanonical("export_report"))
}

func TestIsCanonical_NonCanonicalActions(t *testing.T) {
	assert.False(t, isCanonical("block_order"))
	assert.False(t, isCanonical(""))
	assert.False(t, isCanonical("unknown_action"))
}

// ──── NewEmptyRegistry ─────────────────────────────────────────────────────

func TestNewEmptyRegistry(t *testing.T) {
	reg := NewEmptyRegistry()
	assert.NotNil(t, reg)
	assert.Empty(t, reg.AllowedActions())
}

func TestEmptyRegistry_AllowedActions(t *testing.T) {
	reg := NewEmptyRegistry()
	assert.Empty(t, reg.AllowedActions())
}

func TestEmptyRegistry_GetActionConfig(t *testing.T) {
	reg := NewEmptyRegistry()
	_, ok := reg.GetActionConfig("block_order")
	assert.False(t, ok)
}

func TestEmptyRegistry_IsAllowed(t *testing.T) {
	reg := NewEmptyRegistry()
	assert.False(t, reg.IsAllowed("block_order"))
}

func TestEmptyRegistry_ListActionTypes(t *testing.T) {
	reg := NewEmptyRegistry()
	assert.Empty(t, reg.ListActionTypes())
}

func TestEmptyRegistry_GetActionContract(t *testing.T) {
	reg := NewEmptyRegistry()
	_, ok := reg.GetActionContract("block_order")
	assert.False(t, ok)
}

func TestEmptyRegistry_ValidatePayload(t *testing.T) {
	reg := NewEmptyRegistry()
	errs := reg.ValidatePayload("block_order", map[string]interface{}{})
	assert.Equal(t, []string{"action type not found"}, errs)
}

func TestEmptyRegistry_GetLLMVisibleActions(t *testing.T) {
	reg := NewEmptyRegistry()
	assert.Empty(t, reg.GetLLMVisibleActions())
}

// ──── ActionTypeProviderAdapter ────────────────────────────────────────────

func TestNewActionTypeProviderAdapter(t *testing.T) {
	reg := NewEmptyRegistry()
	adapter := NewActionTypeProviderAdapter(reg)
	assert.NotNil(t, adapter)
}

func TestActionTypeProviderAdapter_ListActionTypes(t *testing.T) {
	reg := NewEmptyRegistry()
	adapter := NewActionTypeProviderAdapter(reg)
	assert.Empty(t, adapter.ListActionTypes())
}

func TestActionTypeProviderAdapter_IsActionAllowed(t *testing.T) {
	reg := NewEmptyRegistry()
	adapter := NewActionTypeProviderAdapter(reg)
	assert.False(t, adapter.IsActionAllowed("block_order"))
}

func TestActionTypeProviderAdapter_GetActionPolicy(t *testing.T) {
	reg := NewEmptyRegistry()
	adapter := NewActionTypeProviderAdapter(reg)
	_, ok := adapter.GetActionPolicy("block_order")
	assert.False(t, ok)
}

// ──── generateTraceID ──────────────────────────────────────────────────────

func TestGenerateTraceID_HasPrefix(t *testing.T) {
	id := generateTraceID()
	assert.Contains(t, id, "trace-")
}

func TestGenerateTraceID_Unique(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()
	assert.NotEqual(t, id1, id2)
}

// ──── buildContract ────────────────────────────────────────────────────────

func TestBuildContract_Basic(t *testing.T) {
	cfg := ActionConfig{
		Description:      "Export report",
		LLMDescription:   "Export a formatted report",
		RiskLevel:        "low",
		RequiresApproval: false,
		Adapter:          "feishu",
	}

	c := buildContract("export_report", cfg)
	assert.Equal(t, "export_report", c.ActionType)
	assert.Equal(t, "Export a formatted report", c.Description)
	assert.Equal(t, "low", c.RiskLevel)
	assert.False(t, c.RequiresReview)
	assert.Equal(t, "feishu", c.Adapter)
	assert.Nil(t, c.RequiredPayload)
	assert.Nil(t, c.PayloadSchema)
}

func TestBuildContract_WithPayloadSchema(t *testing.T) {
	cfg := ActionConfig{
		LLMDescription:   "Notify the data owner",
		RiskLevel:        "medium",
		RequiresApproval: true,
		Adapter:          "feishu",
		PayloadSchemaRaw: map[string]interface{}{
			"type": "object",
			"required": []interface{}{"owner_id", "reason"},
			"properties": map[string]interface{}{
				"owner_id": map[string]interface{}{"type": "string"},
				"reason":   map[string]interface{}{"type": "string"},
			},
		},
	}

	c := buildContract("notify_owner", cfg)
	assert.Equal(t, []string{"owner_id", "reason"}, c.RequiredPayload)
	assert.True(t, c.RequiresReview)
	assert.Contains(t, c.PayloadSchema, "properties")
}

func TestBuildContract_PayloadSchemaNoRequired(t *testing.T) {
	cfg := ActionConfig{
		LLMDescription: "Create a task",
		RiskLevel:      "medium",
		PayloadSchemaRaw: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{"type": "string"},
			},
		},
	}

	c := buildContract("create_followup_task", cfg)
	assert.Nil(t, c.RequiredPayload)
	assert.Contains(t, c.PayloadSchema, "properties")
}

func TestBuildContract_PayloadSchemaRequiredWrongType(t *testing.T) {
	cfg := ActionConfig{
		RiskLevel: "high",
		PayloadSchemaRaw: map[string]interface{}{
			"required": "not_a_slice",
		},
	}

	c := buildContract("some_action", cfg)
	assert.Nil(t, c.RequiredPayload)
	assert.Empty(t, c.PayloadSchema)
}

func TestBuildContract_EmptyActionType(t *testing.T) {
	c := buildContract("", ActionConfig{})
	assert.Equal(t, "", c.ActionType)
	assert.Empty(t, c.Description)
}
