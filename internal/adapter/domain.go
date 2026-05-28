package adapter

// FeishuConfig holds configuration for the Feishu (Lark) dispatch adapter.
type FeishuConfig struct {
	// WebhookURL is the incoming webhook URL for sending messages (legacy).
	WebhookURL string
	// Enabled controls whether the adapter is active.
	Enabled bool
	// AppID is the Feishu app ID for tenant token auth.
	AppID string
	// AppSecret is the Feishu app secret for tenant token auth.
	AppSecret string
	// AppToken is the Feishu bitable app token.
	AppToken string
	// ChatID is the target Feishu chat/group ID.
	ChatID string
}

// GitHubConfig holds configuration for the GitHub dispatch adapter.
type GitHubConfig struct {
	// Token is the GitHub personal access token for API authentication.
	Token string
	// Repo is the repository in "owner/repo" format.
	Repo string
	// Enabled controls whether the adapter is active.
	Enabled bool
}

// CLIConfig holds configuration for the local CLI dispatch adapter.
type CLIConfig struct {
	// LogPath is the filesystem path for the CLI audit log CSV.
	LogPath string
	// Enabled controls whether the adapter is active.
	Enabled bool
}

// ManualConfig holds configuration for the manual review dispatch adapter.
type ManualConfig struct {
	// Enabled controls whether the adapter is active.
	Enabled bool
}

// getString safely extracts a string value from a map with a default fallback.
func getString(m map[string]interface{}, key, defaultValue string) string {
	if m == nil {
		return defaultValue
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

// getFloat extracts a float64 value from a map with a default fallback.
func getFloat(m map[string]interface{}, key string, defaultVal float64) float64 {
	if m == nil {
		return defaultVal
	}
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return defaultVal
}

// ActionChannel returns the dispatch channel for a given action type.
func ActionChannel(actionType string) string {
	switch actionType {
	case "export_report", "notify_owner", "create_outbox_message":
		return "feishu"
	case "create_followup_task":
		return "github"
	default:
		return "unknown"
	}
}
