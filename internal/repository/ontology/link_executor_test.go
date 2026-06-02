package ontology

import (
	"testing"
)

func TestIsValidSort_ValidExpressions(t *testing.T) {
	tests := []struct {
		name  string
		sort  string
		valid bool
	}{
		{"empty string", "", true},
		{"simple column", "column_name", true},
		{"column with underscores", "order_purchase_timestamp", true},
		{"column ASC", "column ASC", true},
		{"column DESC", "column DESC", true},
		{"column asc", "column asc", true},
		{"column desc", "column desc", true},
		{"multi-column", "col1, col2", true},
		{"multi-column with directions", "col1 ASC, col2 DESC", true},
		{"multi-column underscore", "seller_state, seller_city", true},
		{"column with numeric suffix", "col_123", true},
		{"leading underscore", "_private_col", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSort(tt.sort)
			if got != tt.valid {
				t.Errorf("isValidSort(%q) = %v, want %v", tt.sort, got, tt.valid)
			}
		})
	}
}

func TestIsValidSort_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name string
		sort string
	}{
		{"SQL injection with semicolon", "1; DROP TABLE users"},
		{"SQL injection with UNION", "id; SELECT * FROM users"},
		{"parenthesized expression", "(SELECT 1)"},
		{"function call", "COUNT(*)"},
		{"subquery", "(SELECT max(id) FROM t)"},
		{"string literal", "'foo'"},
		{"numeric literal", "12345"},
		{"quoted identifier", `"column_name"`},
		{"wildcard star", "*"},
		{"dash in name", "column-name"},
		{"dot notation", "schema.table"},
		{"backtick quoted", "`column`"},
		{"newline injection", "col\nDROP TABLE t"},
		{"tab injection", "col\tDESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSort(tt.sort)
			if got {
				t.Errorf("isValidSort(%q) = true, want false", tt.sort)
			}
		})
	}
}
