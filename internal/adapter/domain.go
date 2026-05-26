package adapter

// FeishuConfig holds configuration for the Feishu (Lark) dispatch adapter.
type FeishuConfig struct {
	// WebhookURL is the incoming webhook URL for sending messages.
	WebhookURL string
	// Enabled controls whether the adapter is active.
	Enabled bool
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
