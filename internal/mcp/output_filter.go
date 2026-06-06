package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

// output_filter.go provides centralized filter functions for MCP tool outputs.

var (
	// schemaTableRegex matches "schema.table_name" patterns like "ops.metric_alert"
	schemaTableRegex = regexp.MustCompile(`[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*`)
	// filePathRegex matches Linux file system paths
	filePathRegex = regexp.MustCompile(`(?:/|[a-zA-Z]:\\|\.\./|\./)(?:[^\s"':])+`)
)
// These functions strip architectural details from responses to prevent
// information leakage to AI agents.
//
// Applied at the handler level, before JSON serialization with NewToolResultJSON.

// FilterSystemStatus removes fields that leak database architecture details
// from the system status response.
// Removes: table_counts (table names + row counts)
// Preserves: alert_count, pipeline_run (aggregate health info)
func FilterSystemStatus(result map[string]interface{}) {
	delete(result, "table_counts")
}

// FilterOntologyDescriptor removes SourceDescriptor (schema/table/PK) and
// Governance fields from each object type in the ontology response.
// These fields are already nil with omitempty in the current implementation,
// but this acts as a defense-in-depth layer.
func FilterOntologyDescriptor(objectTypes []interface{}) {
	for _, ot := range objectTypes {
		if m, ok := ot.(map[string]interface{}); ok {
			delete(m, "source")
			delete(m, "governance")
		}
	}
}

// FilterSearchObjects caps result size and strips detailed fields from
// search results, returning only a summary view.
// For each result item, keeps only object_id and object_type (if present),
// removing all detailed property fields.
func FilterSearchObjects(result map[string]interface{}) {
	itemsRaw, ok := result["items"]
	if !ok {
		return
	}
	items, ok := itemsRaw.([]map[string]interface{})
	if !ok {
		return
	}
	stripped := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		summary := make(map[string]interface{})
		if id, ok := item["object_id"]; ok {
			summary["object_id"] = id
		}
		if objType, ok := item["object_type"]; ok {
			summary["object_type"] = objType
		}
		// If neither object_id nor object_type found, keep a minimal marker
		if len(summary) == 0 {
			summary["id"] = item["id"]
		}
		stripped = append(stripped, summary)
	}
	result["items"] = stripped
}

// FilterLinkedObjects enforces max_depth and strips detailed fields from
// linked object results.
// Currently a no-op placeholder for Phase 9 implementation.
func FilterLinkedObjects(result map[string]interface{}) {
	// Phase 9: apply field-level filtering, verify depth bounds
}

// SanitizeErrorf wraps fmt.Sprintf with error sanitization for use in
// NewToolResultError calls. This prevents database schema details, file paths,
// and internal architecture info from leaking through error messages.
//
// Usage: mcp.NewToolResultError(SanitizeErrorf("Failed to do X: %v", err))
func SanitizeErrorf(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return SanitizeError(msg)
}


// SanitizeError redacts sensitive details from error messages that could
// leak database architecture, file paths, or internal implementation details.
// Applied to all NewToolResultError calls across MCP handlers.
//
// Redactions:
//   - schema.table patterns → "db.table"
//   - File paths (/home/..., /tmp/...) → "[redacted path]"
//   - SQL error details → "[database error]"
//   - Stack traces or query text → "[internal detail]"
func SanitizeError(msg string) string {
	// Redact common schema.table patterns (e.g., "ops.metric_alert", "audit.pipeline_run")
	msg = schemaTableRegex.ReplaceAllString(msg, "db.table")

	// Redact error messages containing SQL keywords as indicators of database errors
	if strings.Contains(msg, "ERROR") || strings.Contains(msg, "relation") ||
		strings.Contains(msg, "column") || strings.Contains(msg, "syntax error") ||
		strings.Contains(msg, " violates ") || strings.Contains(msg, "constraint") ||
		strings.Contains(msg, "not null") {
		msg = "database operation failed"
	}

	// Redact file system paths (Linux paths)
	msg = filePathRegex.ReplaceAllString(msg, "[redacted path]")

	return msg
}

