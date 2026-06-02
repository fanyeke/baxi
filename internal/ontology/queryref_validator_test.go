package ontology

import (
	"strings"
	"testing"
)

func TestValidateQueryRef_AllowsValidSelect(t *testing.T) {
	tests := []string{
		"SELECT * FROM dwd.orders WHERE id = $1",
		"select * from mart.items where id = $1",
		"  SELECT * FROM ops.tasks WHERE task_id = $1  ",
		"SELECT a, b, c FROM dwd.events WHERE event_id = $1 AND status = 'active'",
		"SELECT * FROM mart.orders WHERE id = $1 LIMIT 10",
		"SELECT * FROM mart.orders o JOIN dwd.items i ON o.item_id = i.id WHERE o.id = $1",
		"SELECT * FROM mart.orders WHERE id = $1 ORDER BY created_at DESC",
	}
	for _, tmpl := range tests {
		t.Run(tmpl, func(t *testing.T) {
			err := ValidateQueryRef(tmpl)
			if err != nil {
				t.Errorf("ValidateQueryRef(%q) unexpected error: %v", tmpl, err)
			}
		})
	}
}

func TestValidateQueryRef_RequiresSelectStart(t *testing.T) {
	tests := []struct {
		tmpl         string
		expectErrMsg string // substring to look for in the error
	}{
		{"", ""}, // empty template: any error is acceptable
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", "must start with SELECT"},
		{"INSERT INTO dwd.orders VALUES ($1)", "forbidden DML keyword"},   // security check fires first
		{"   UPDATE mart.items SET x=1 WHERE id=$1   ", "forbidden DML keyword"}, // security check fires first
		{"EXPLAIN SELECT * FROM dwd.orders WHERE id = $1", "must start with SELECT"},
	}
	for _, tt := range tests {
		t.Run(tt.tmpl, func(t *testing.T) {
			err := ValidateQueryRef(tt.tmpl)
			if err == nil {
				t.Errorf("expected error for template %q", tt.tmpl)
			}
			if tt.expectErrMsg != "" && !strings.Contains(err.Error(), tt.expectErrMsg) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErrMsg, err)
			}
		})
	}
}

func TestValidateQueryRef_RequiresParam(t *testing.T) {
	tests := []string{
		"SELECT * FROM dwd.orders WHERE id = 123",
		"SELECT * FROM dwd.orders",
		"SELECT * FROM dwd.orders WHERE id = $2",
	}
	for _, tmpl := range tests {
		t.Run(tmpl, func(t *testing.T) {
			err := ValidateQueryRef(tmpl)
			if err == nil {
				t.Errorf("expected error for template without $1: %q", tmpl)
			}
			if !strings.Contains(err.Error(), "$1") {
				t.Errorf("expected error to mention $1, got: %v", err)
			}
		})
	}
}

func TestValidateQueryRef_RejectsDML(t *testing.T) {
	tests := []struct {
		tmpl    string
		keyword string
	}{
		{"INSERT INTO dwd.orders VALUES ($1)", "INSERT"},
		{"UPDATE mart.items SET name='x' WHERE id=$1", "UPDATE"},
		{"DELETE FROM dwd.orders WHERE id=$1", "DELETE"},
		{"DROP TABLE dwd.orders", "DROP"},
		{"ALTER TABLE mart.items ADD COLUMN x INT", "ALTER"},
		{"TRUNCATE TABLE dwd.orders", "TRUNCATE"},
		{"EXECUTE some_proc($1)", "EXECUTE"},
		{"COPY dwd.orders TO '/tmp/out'", "COPY"},
	}
	for _, tt := range tests {
		t.Run(tt.keyword, func(t *testing.T) {
			err := ValidateQueryRef(tt.tmpl)
			if err == nil {
				t.Errorf("expected error for DML template %q", tt.tmpl)
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

func TestValidateQueryRef_RejectsSemicolon(t *testing.T) {
	tests := []string{
		"SELECT * FROM dwd.orders WHERE id = $1; DROP TABLE dwd.orders",
		"SELECT * FROM dwd.orders WHERE id = $1;",
		";SELECT * FROM dwd.orders WHERE id = $1",
	}
	for _, tmpl := range tests {
		t.Run(tmpl, func(t *testing.T) {
			err := ValidateQueryRef(tmpl)
			if err == nil {
				t.Errorf("expected error for template with semicolon: %q", tmpl)
			}
			if !strings.Contains(err.Error(), "semicolon") {
				t.Errorf("expected semicolon error, got: %v", err)
			}
		})
	}
}

func TestValidateQueryRef_RejectsComments(t *testing.T) {
	tests := []string{
		"SELECT * FROM dwd.orders WHERE id = $1 -- comment",
		"/* malicious */ SELECT * FROM dwd.orders WHERE id = $1",
		"SELECT * FROM dwd.orders WHERE id = $1 /* inline */",
	}
	for _, tmpl := range tests {
		t.Run(tmpl, func(t *testing.T) {
			err := ValidateQueryRef(tmpl)
			if err == nil {
				t.Errorf("expected error for template with comment: %q", tmpl)
			}
			if !strings.Contains(err.Error(), "comment") {
				t.Errorf("expected comment error, got: %v", err)
			}
		})
	}
}

func TestValidateQueryRef_RejectsDisallowedSchema(t *testing.T) {
	tests := []string{
		"SELECT * FROM public.users WHERE id = $1",
		"SELECT * FROM dwd.orders JOIN pg_catalog.pg_class c ON c.oid = $1",
		"SELECT * FROM information_schema.tables WHERE table_name = $1",
	}
	for _, tmpl := range tests {
		t.Run(tmpl, func(t *testing.T) {
			err := ValidateQueryRef(tmpl)
			if err == nil {
				t.Errorf("expected error for template with disallowed schema: %q", tmpl)
			}
			if !strings.Contains(err.Error(), "disallowed schema") {
				t.Errorf("expected disallowed schema error, got: %v", err)
			}
		})
	}
}

func TestValidateQueryRef_AllowsAllThreeSchemas(t *testing.T) {
	tests := []string{
		"SELECT * FROM dwd.orders WHERE id = $1",
		"SELECT * FROM mart.products WHERE id = $1",
		"SELECT * FROM ops.jobs WHERE id = $1",
	}
	for _, tmpl := range tests {
		t.Run(tmpl, func(t *testing.T) {
			err := ValidateQueryRef(tmpl)
			if err != nil {
				t.Errorf("ValidateQueryRef(%q) unexpected error: %v", tmpl, err)
			}
		})
	}
}
