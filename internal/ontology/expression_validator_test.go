package ontology

import (
	"strings"
	"testing"
)

func TestValidateExpression_AllowsSimpleExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"empty string", "", false},
		{"arithmetic", "price * 1.2", false},
		{"simple column reference", "review_score", false},
		{"column with table qualifier", "order_level.review_score", false},
		{"avg fragment", "AVG(review_score)", false},
		{"coalesce fragment", "COALESCE(review_score, 0)", false},
		{"nested functions", "ROUND(AVG(review_score), 2)", false},
		{"case expression", "CASE WHEN status='active' THEN 1 ELSE 0 END", false},
		{"cast expression", "CAST(price AS FLOAT)", false},
		{"nullif guard", "NULLIF(review_score, -1)", false},
		{"count star", "COUNT(*)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExpression(%q) error = %v, wantErr = %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestValidateExpression_RejectsSemicolon(t *testing.T) {
	tests := []string{
		"AVG(review_score); DROP TABLE users",
		";",
		"SELECT 1;",
		"SELECT 1; SELECT 2",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			err := ValidateExpression(expr)
			if err == nil {
				t.Errorf("expected error for expression with semicolon: %q", expr)
			}
			if !strings.Contains(err.Error(), "semicolon") {
				t.Errorf("expected semicolon error, got: %v", err)
			}
		})
	}
}

func TestValidateExpression_RejectsComments(t *testing.T) {
	tests := []string{
		"AVG(review_score) -- comment",
		"/* malicious */ SELECT 1",
		"-- inline comment\nSELECT 1",
		"SELECT 1 /* inline block */ FROM t",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			err := ValidateExpression(expr)
			if err == nil {
				t.Errorf("expected error for expression with comment: %q", expr)
			}
			if !strings.Contains(err.Error(), "comment") {
				t.Errorf("expected comment error, got: %v", err)
			}
		})
	}
}

func TestValidateExpression_RejectsDML(t *testing.T) {
	tests := []struct {
		expr    string
		keyword string
	}{
		{"INSERT INTO users VALUES (1)", "INSERT"},
		{"UPDATE users SET name='x'", "UPDATE"},
		{"DELETE FROM users WHERE id=1", "DELETE"},
		{"DROP TABLE users", "DROP"},
		{"ALTER TABLE users ADD COLUMN x INT", "ALTER"},
		{"TRUNCATE TABLE users", "TRUNCATE"},
		{"EXECUTE some_proc()", "EXECUTE"},
		{"COPY users TO '/tmp/out'", "COPY"},
	}
	for _, tt := range tests {
		t.Run(tt.keyword, func(t *testing.T) {
			err := ValidateExpression(tt.expr)
			if err == nil {
				t.Errorf("expected error for DML expression %q", tt.expr)
			}
			if !strings.Contains(err.Error(), "forbidden DML keyword") {
				t.Errorf("expected DML keyword error, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.keyword) {
				t.Errorf("expected error to mention keyword %q, got: %v", tt.keyword, err)
			}
		})
	}
}

func TestValidateExpression_AllowsWhitelistFunctions(t *testing.T) {
	tests := []string{
		"AVG(price)",
		"SUM(quantity)",
		"COUNT(DISTINCT user_id)",
		"MIN(created_at)",
		"MAX(amount)",
		"COALESCE(margin, 0)",
		"NULLIF(denominator, 0)",
		"CAST(price AS FLOAT)",
		"ROUND(avg_price, 2)",
		"GREATEST(a, b)",
		"LEAST(a, b)",
		"ABS(delta)",
		"CONCAT(first_name, ' ', last_name)",
		"UPPER(status)",
		"LOWER(email)",
		"LENGTH(description)",
		"SUBSTRING(title, 1, 100)",
		"TRIM(notes)",
		"DATE_TRUNC('month', created_at)",
		"EXTRACT(YEAR FROM created_at)",
		"NOW()",
		"CURRENT_DATE",
		"CURRENT_TIMESTAMP",
		"CASE WHEN a > 0 THEN 'pos' ELSE 'neg' END",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			err := ValidateExpression(expr)
			if err != nil {
				t.Errorf("unexpected error for whitelisted function %q: %v", expr, err)
			}
		})
	}
}

func TestValidateExpression_AllowsSubquery(t *testing.T) {
	tests := []string{
		"(SELECT AVG(review_score) FROM reviews WHERE seller_id = ?)",
		"(SELECT COUNT(*) FROM orders WHERE status = 'completed')",
		"(SELECT COALESCE(SUM(amount), 0) FROM payments WHERE order_id = ?)",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			err := ValidateExpression(expr)
			if err != nil {
				t.Errorf("unexpected error for subquery %q: %v", expr, err)
			}
		})
	}
}

func TestValidateExpression_RejectsBlacklistedKeywords(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"non-whitelisted function", "SLEEP(10)"},
		{"pg function", "PG_SLEEP(5)"},
		{"system function", "CURRENT_USER()"},
		{"set statement", "SET search_path TO public"},
		{"grant statement", "GRANT SELECT ON users TO attacker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpression(tt.expr)
			if err == nil {
				t.Errorf("expected error for expression: %q", tt.expr)
			}
		})
	}
}

func TestValidateExpression_NontrivialSelectStatement(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"select with from", "SELECT AVG(price) FROM products"},
		{"select with where", "SELECT COUNT(*) FROM orders WHERE status = 'active'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpression(tt.expr)
			if err != nil {
				t.Errorf("unexpected error for valid SELECT statement %q: %v", tt.expr, err)
			}
		})
	}
}

func TestValidateExpression_RejectsNonSelectStatement(t *testing.T) {
	tests := []string{
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"EXPLAIN SELECT 1 FROM users",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			err := ValidateExpression(expr)
			if err == nil {
				t.Errorf("expected error for non-SELECT statement: %q", expr)
			}
		})
	}
}
