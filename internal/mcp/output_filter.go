package mcp

// output_filter.go provides centralized filter functions for MCP tool outputs.
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
// Currently a no-op placeholder for Phase 10 implementation.
func FilterSearchObjects(result map[string]interface{}) {
	// Phase 10: cap limit, strip detailed fields from items
}

// FilterLinkedObjects enforces max_depth and strips detailed fields from
// linked object results.
// Currently a no-op placeholder for Phase 9 implementation.
func FilterLinkedObjects(result map[string]interface{}) {
	// Phase 9: apply field-level filtering, verify depth bounds
}
