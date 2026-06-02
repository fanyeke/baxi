package llm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── NewRepairPromptRenderer ──────────────────────────────────────────────

func TestNewRepairPromptRenderer_Success(t *testing.T) {
	r, err := NewRepairPromptRenderer()
	require.NoError(t, err)
	require.NotNil(t, r)
}

// ──── RenderRepairPrompt ───────────────────────────────────────────────────

func TestRenderRepairPrompt_NoErrors(t *testing.T) {
	r, err := NewRepairPromptRenderer()
	require.NoError(t, err)

	result, err := r.RenderRepairPrompt(nil)
	require.NoError(t, err)

	assert.Contains(t, result, "Decision Repair Prompt")
	assert.Contains(t, result, "Validation Errors")
	assert.NotContains(t, result, "- :")
}

func TestRenderRepairPrompt_SingleError(t *testing.T) {
	r, err := NewRepairPromptRenderer()
	require.NoError(t, err)

	errors := []ValidationError{
		{Field: "action_type", Message: "action type is not in allowed_actions"},
	}

	result, err := r.RenderRepairPrompt(errors)
	require.NoError(t, err)

	assert.Contains(t, result, "action_type")
	assert.Contains(t, result, "action type is not in allowed_actions")
	assert.Contains(t, result, "- action_type: action type is not in allowed_actions")
}

func TestRenderRepairPrompt_MultipleErrors(t *testing.T) {
	r, err := NewRepairPromptRenderer()
	require.NoError(t, err)

	errors := []ValidationError{
		{Field: "action_type", Message: "not in allowed_actions"},
		{Field: "confidence", Message: "must be between 0 and 1"},
	}

	result, err := r.RenderRepairPrompt(errors)
	require.NoError(t, err)

	assert.Contains(t, result, "action_type")
	assert.Contains(t, result, "confidence")
	assert.Contains(t, result, "not in allowed_actions")
	assert.Contains(t, result, "must be between 0 and 1")

	// Verify exactly two error lines
	lines := strings.Split(strings.TrimSpace(result), "\n")
	errorLines := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			errorLines++
		}
	}
	assert.Equal(t, 2, errorLines)
}

func TestRenderRepairPrompt_OutputContainsHeaderAndInstructions(t *testing.T) {
	r, err := NewRepairPromptRenderer()
	require.NoError(t, err)

	errors := []ValidationError{
		{Field: "test", Message: "test error"},
	}

	result, err := r.RenderRepairPrompt(errors)
	require.NoError(t, err)

	assert.Contains(t, result, "Decision Repair Prompt")
	assert.Contains(t, result, "Validation Errors")
	assert.Contains(t, result, "Original Context")
	assert.Contains(t, result, "Instructions")
}

func TestRenderRepairPrompt_NoErrorsRendersEmptyErrors(t *testing.T) {
	r, err := NewRepairPromptRenderer()
	require.NoError(t, err)

	result, err := r.RenderRepairPrompt([]ValidationError{})
	require.NoError(t, err)

	assert.Contains(t, result, "Decision Repair Prompt")
	assert.NotContains(t, result, "- :")
}
