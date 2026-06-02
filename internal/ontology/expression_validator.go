package ontology

import (
	"fmt"
	"regexp"
	"strings"
)

// whitelistFunctions is the set of SQL functions allowed in expressions.
var whitelistFunctions = map[string]bool{
	"AVG": true, "SUM": true, "COUNT": true, "MIN": true, "MAX": true,
	"CASE": true, "COALESCE": true, "NULLIF": true, "CAST": true,
	"ROUND": true, "GREATEST": true, "LEAST": true, "ABS": true,
	"CONCAT": true, "UPPER": true, "LOWER": true, "LENGTH": true,
	"SUBSTRING": true, "TRIM": true, "DATE_TRUNC": true, "EXTRACT": true,
	"NOW": true, "CURRENT_DATE": true, "CURRENT_TIMESTAMP": true,
}

// dmlKeywords lists DML/DDL keywords that are forbidden in expressions.
var dmlKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "TRUNCATE", "EXECUTE", "COPY",
}

// forbiddenStatementKeywords lists additional SQL keywords that indicate a
// non-expression statement and are forbidden in YAML expression fields.
var forbiddenStatementKeywords = []string{
	"SET ", "GRANT ", "REVOKE ", "CREATE ", "BEGIN ", "COMMIT ", "ROLLBACK ",
}

// clauseKeywords lists SQL clause keywords that indicate a full statement.
// Each includes a trailing space to avoid matching column names.
var clauseKeywords = []string{
	"FROM ", "WHERE ", "GROUP ", "ORDER ", "HAVING ",
	"JOIN ", "INNER ", "LEFT ", "RIGHT ", "OUTER ",
}

// extractFromPattern matches EXTRACT(... FROM ...) expressions so the interior
// FROM keyword is not mistaken for a SQL FROM clause.
var extractFromPattern = regexp.MustCompile(`(?i)EXTRACT\s*\([^)]*\bFROM\b[^)]*\)`)

var commentPattern = regexp.MustCompile(`/\*.*?\*/|--[^\n]*`)

// ValidateExpression checks an SQL expression string embedded in YAML
// configuration (ObjectPropertyV2.Expression) for safety and correctness.
//
// Checks performed:
//   - Empty string is allowed (returns nil).
//   - Semicolons are rejected (prevents multi-statement injection).
//   - SQL comments (-- and /* */) are rejected.
//   - DML/DDL keywords (INSERT, UPDATE, DELETE, DROP, ALTER, TRUNCATE,
//     EXECUTE, COPY) are rejected.
//   - Expressions that look like full statements must begin with SELECT.
//   - Only whitelisted SQL functions may be used.
func ValidateExpression(expr string) error {
	if expr == "" {
		return nil
	}

	// Reject semicolons — prevents multiple statements.
	if strings.Contains(expr, ";") {
		return fmt.Errorf("expression contains semicolon (not allowed)")
	}

	// Reject SQL comments.
	if commentPattern.MatchString(expr) {
		return fmt.Errorf("expression contains SQL comments (not allowed)")
	}

	exprUpper := strings.ToUpper(expr)

	// Reject DML/DDL keywords.
	for _, kw := range dmlKeywords {
		if strings.Contains(exprUpper, kw) {
			return fmt.Errorf("expression contains forbidden DML keyword: %s", kw)
		}
	}

	// Reject forbidden statement keywords (SET, GRANT, REVOKE, CREATE, etc.).
	for _, kw := range forbiddenStatementKeywords {
		if strings.Contains(exprUpper, kw) {
			return fmt.Errorf("expression contains forbidden statement keyword: %s", strings.TrimSpace(kw))
		}
	}

	trimmed := strings.TrimSpace(expr)

	// If the expression contains SQL clause keywords it appears to be a
	// full statement — it must begin with SELECT (possibly wrapped in parens
	// for a subquery).  Mask EXTRACT(... FROM ...) expressions first, so the
	// interior FROM keyword is not mistaken for a SQL FROM clause.
	maskedForClauseCheck := extractFromPattern.ReplaceAllString(exprUpper, "EXTRACT_MASKED")
	if containsClauseKeyword(maskedForClauseCheck) {
		if strings.HasPrefix(trimmed, "(") {
			inner := strings.TrimSpace(trimmed[1:])
			if len(inner) > 0 && !strings.HasPrefix(strings.ToUpper(inner), "SELECT") {
				return fmt.Errorf("subquery must start with SELECT")
			}
		} else if !strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") {
			return fmt.Errorf("statement expression must start with SELECT")
		}
	}

	// Validate that only whitelisted functions are used.
	return validateFunctions(exprUpper)
}

// containsClauseKeyword returns true if the expression contains a SQL clause
// keyword that indicates it is a full statement rather than a simple fragment.
func containsClauseKeyword(exprUpper string) bool {
	for _, kw := range clauseKeywords {
		if strings.Contains(exprUpper, kw) {
			return true
		}
	}
	return false
}

// validateFunctions checks that every function call in expr uses a
// whitelisted function name.
func validateFunctions(exprUpper string) error {
	re := regexp.MustCompile(`\b([A-Z_][A-Z0-9_]*)\s*\(`)
	matches := re.FindAllStringSubmatch(exprUpper, -1)
	for _, m := range matches {
		funcName := m[1]
		if !whitelistFunctions[funcName] {
			return fmt.Errorf("expression uses non-whitelisted function: %s", funcName)
		}
	}
	return nil
}
