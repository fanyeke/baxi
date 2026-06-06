package mcp

import "os"

// Default MCP server identity values for information containment.
// These replace the Baxi-specific server identity to prevent
// AI agents from inferring project architecture from server info.
const (
	defaultServerName    = "Data Processing Server"
	defaultServerVersion = "1.0.0"
	defaultInstructions  = "Platform for data analysis and action management"
)

// getServerName returns the MCP server name.
// Override via MCP_SERVER_NAME env var.
// Default: "Data Processing Server" (generic, no project identity).
func getServerName() string {
	if v := os.Getenv("MCP_SERVER_NAME"); v != "" {
		return v
	}
	return defaultServerName
}

// getServerVersion returns the MCP server version.
// Override via MCP_SERVER_VERSION env var.
func getServerVersion() string {
	if v := os.Getenv("MCP_SERVER_VERSION"); v != "" {
		return v
	}
	return defaultServerVersion
}

// getServerInstructions returns the MCP server instructions text.
// Override via MCP_SERVER_INSTRUCTIONS env var.
// Default: generic description that does not reveal project identity.
func getServerInstructions() string {
	if v := os.Getenv("MCP_SERVER_INSTRUCTIONS"); v != "" {
		return v
	}
	return defaultInstructions
}

// isLegacyToolsEnabled returns true if legacy tool name aliases should be registered.
// When enabled, old tool names are registered alongside new names for backward
// compatibility with existing Pi Agent integrations.
// Controlled by MCP_ENABLE_LEGACY_TOOLS env var (default: "true" for safe transition).
func isLegacyToolsEnabled() bool {
	return os.Getenv("MCP_ENABLE_LEGACY_TOOLS") != "false"
}
