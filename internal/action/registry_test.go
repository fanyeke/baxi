package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestRegistryYAML(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "action_registry.yml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func defaultTestYAML() string {
	return `
actions:
  create_feishu_report:
    description: "创建飞书日报/异常报告"
    risk_level: low
    requires_approval: false
    allowed_by: [business_ops, seller_ops]
  notify_owner:
    description: "通知负责人处理异常"
    risk_level: low
    requires_approval: false
    allowed_by: [business_ops, seller_ops]
  create_followup_task:
    description: "创建跟进任务"
    risk_level: medium
    requires_approval: true
    allowed_by: [business_ops, seller_ops]
  recommend_business_strategy:
    description: "推荐业务策略"
    risk_level: medium
    requires_approval: true
    allowed_by: [business_ops, category_ops]
  modify_business_policy:
    description: "修改业务策略或规则"
    risk_level: high
    requires_approval: true
    allowed_by: [business_ops]
`
}

func TestNewActionRegistry_LoadsYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())

	reg, err := NewActionRegistry(path)
	require.NoError(t, err)
	require.NotNil(t, reg)
}

func TestNewActionRegistry_DefaultPath(t *testing.T) {
	// Must fail since the default path won't exist in test temp dir
	_, err := NewActionRegistry("")
	assert.Error(t, err)
}

func TestIsAllowed_CanonicalActions(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	for _, action := range CanonicalActions {
		t.Run(action, func(t *testing.T) {
			assert.True(t, reg.IsAllowed(action), "expected %s to be allowed", action)
		})
	}
}

func TestIsAllowed_NonWhitelistedActions(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	forbidden := []string{"hack_database", "delete_all", "create_feishu_report", "recommend_business_strategy", "modify_business_policy"}
	for _, action := range forbidden {
		t.Run(action, func(t *testing.T) {
			assert.False(t, reg.IsAllowed(action), "expected %s to be rejected", action)
		})
	}
}

func TestGetActionConfig_KnownAction(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	cfg, ok := reg.GetActionConfig("notify_owner")
	assert.True(t, ok)
	assert.Equal(t, "通知负责人处理异常", cfg.Description)
	assert.Equal(t, "low", cfg.RiskLevel)
	assert.False(t, cfg.RequiresApproval)
	assert.Equal(t, []string{"business_ops", "seller_ops"}, cfg.AllowedBy)
}

func TestGetActionConfig_MissingActionGetsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	// export_report is whitelisted but not in the test YAML → should get defaults
	cfg, ok := reg.GetActionConfig("export_report")
	assert.True(t, ok)
	assert.Equal(t, "", cfg.Description)
	assert.Equal(t, "medium", cfg.RiskLevel)
	assert.True(t, cfg.RequiresApproval)
	assert.Empty(t, cfg.AllowedBy)

	// create_outbox_message is also missing from YAML
	cfg, ok = reg.GetActionConfig("create_outbox_message")
	assert.True(t, ok)
	assert.Equal(t, "", cfg.Description)
	assert.Equal(t, "medium", cfg.RiskLevel)
	assert.True(t, cfg.RequiresApproval)
	assert.Empty(t, cfg.AllowedBy)
}

func TestGetActionConfig_UnknownAction(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	_, ok := reg.GetActionConfig("hack_database")
	assert.False(t, ok)
}

func TestGetActionConfig_NonWhitelistedActionInYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	// create_feishu_report is in YAML but not whitelisted
	_, ok := reg.GetActionConfig("create_feishu_report")
	assert.False(t, ok)
}

func TestAllowedActions_ReturnsExactlyFour(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	actions := reg.AllowedActions()
	assert.Len(t, actions, 4)
	assert.ElementsMatch(t, CanonicalActions, actions)
}

func TestReload_Succeeds(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	// Reload should work without error
	err = reg.Reload()
	assert.NoError(t, err)

	// Verify still works after reload
	assert.True(t, reg.IsAllowed("notify_owner"))
	assert.False(t, reg.IsAllowed("hack_database"))
}

func TestReload_FailsOnBadFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, defaultTestYAML())
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	// Overwrite with invalid YAML
	badPath := filepath.Join(dir, "action_registry.yml")
	err = os.WriteFile(badPath, []byte("invalid: [yaml: broken"), 0644)
	require.NoError(t, err)

	err = reg.Reload()
	assert.Error(t, err)
}

func TestNewActionRegistry_FailsOnBadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yml")
	err := os.WriteFile(path, []byte("{{invalid yaml}}"), 0644)
	require.NoError(t, err)

	_, err = NewActionRegistry(path)
	assert.Error(t, err)
}

func TestNewActionRegistry_FailsOnMissingFile(t *testing.T) {
	_, err := NewActionRegistry("/nonexistent/path/action_registry.yml")
	assert.Error(t, err)
}
