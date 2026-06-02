package ontology

import (
	"fmt"
	"regexp"
	"strings"
)

// allowedSchemas lists the schema names permitted in query_ref SQL templates.
var allowedSchemas = map[string]bool{
	"dwd":  true,
	"mart": true,
	"ops":  true,
}

// dmlKeywordsQueryRef lists DML/DDL keywords forbidden in query_ref templates.
var dmlKeywordsQueryRef = []string{
	"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "TRUNCATE", "EXECUTE", "COPY",
}

// schemaTablePattern matches schema-qualified table references in FROM and JOIN
// clauses, e.g. "FROM dwd.orders" or "JOIN mart.items".
var schemaTablePattern = regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+["]?([a-zA-Z_][a-zA-Z0-9_]*)["]?\s*\.`)

// ValidateQueryRef checks a query_ref SQL template for safety and correctness.
//
// Checks performed:
//   - Must start with SELECT (case-insensitive, leading whitespace trimmed).
//   - Must contain the $1 parameter placeholder for the source object ID.
//   - Semicolons are rejected (prevents multi-statement injection).
//   - DML/DDL keywords (INSERT, UPDATE, DELETE, DROP, ALTER, TRUNCATE,
//     EXECUTE, COPY) are rejected.
//   - SQL comments (-- and /* */) are rejected.
//   - Any schema-qualified table references (schema.table) must use one of
//     the allowed schemas: dwd, mart, ops.
//
// The caller (compileQueryRef) appends LIMIT/OFFSET if the template does not
// already contain a LIMIT clause.
func ValidateQueryRef(template string) error {
	if template == "" {
		return fmt.Errorf("query_ref template must not be empty")
	}

	// 1. SECURITY CHECKS FIRST — catch malicious input before format validation.

	// Reject semicolons — prevents multiple statements.
	if strings.Contains(template, ";") {
		return fmt.Errorf("query_ref template contains semicolon (not allowed)")
	}

	// Reject DML/DDL keywords.
	upper := strings.ToUpper(template)
	for _, kw := range dmlKeywordsQueryRef {
		if strings.Contains(upper, kw) {
			return fmt.Errorf("query_ref template contains forbidden DML keyword: %s", kw)
		}
	}

	// Reject SQL comments.
	if commentPattern.MatchString(template) {
		return fmt.Errorf("query_ref template contains SQL comments (not allowed)")
	}

	// 2. FORMAT CHECKS — after security, validate structure.
	trimmed := strings.TrimSpace(template)

	// Must start with SELECT (case-insensitive).
	if !strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") {
		return fmt.Errorf("query_ref template must start with SELECT")
	}

	// Must contain $1 parameter placeholder.
	if !strings.Contains(template, "$1") {
		return fmt.Errorf("query_ref template must contain $1 parameter placeholder")
	}

	// Validate schema references.
	if err := validateQueryRefSchemas(template); err != nil {
		return err
	}

	return nil
}

// validateQueryRefSchemas checks that any schema-qualified table references
// in the template use only allowed schemas (dwd, mart, ops).
func validateQueryRefSchemas(template string) error {
	matches := schemaTablePattern.FindAllStringSubmatch(template, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		schema := strings.ToLower(m[1])
		if !allowedSchemas[schema] {
			return fmt.Errorf("query_ref template references disallowed schema %q (only dwd, mart, ops are permitted)", schema)
		}
	}
	return nil
}
